package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

func TestExtractUsername(t *testing.T) {
	type tcase struct {
		name          string
		token         *jwt.Token
		error         error
		username      string
		isBuiltInUser bool
	}
	tcases := []tcase{
		{
			name:          "oidc user",
			token:         jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "some_user@email.com", "iss": "external_issuer"}),
			error:         nil,
			username:      "some_user@email.com",
			isBuiltInUser: false,
		},
		{
			name:          "built-in user",
			token:         jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "admin:login", "iss": "everest"}),
			error:         nil,
			username:      "admin",
			isBuiltInUser: true,
		},
		{
			name:          "no sub in token",
			token:         jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{}),
			error:         errExtractSub,
			username:      "",
			isBuiltInUser: false,
		},
		{
			name:          "no iss in token",
			token:         jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "smth"}),
			error:         errExtractIss,
			username:      "",
			isBuiltInUser: false,
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			username, isBuiltInUser, err := extractUsername(tc.token)
			assert.Equal(t, username, tc.username)
			assert.Equal(t, isBuiltInUser, tc.isBuiltInUser)
			assert.Equal(t, tc.error, err)
		})
	}
}

func TestExtractIssueTime(t *testing.T) {
	type tcase struct {
		name  string
		token *jwt.Token
		error error
		time  *time.Time
	}
	tcases := []tcase{
		{
			name:  "no iat field",
			token: jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{}),
			error: errExtractIssueTime,
			time:  nil,
		},
		{
			name:  "wrong iat field",
			token: jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"iat": "sdfssfsf"}),
			error: errExtractIssueTime,
			time:  nil,
		},
		{
			name:  "valid iat field",
			token: jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"iat": float64(1747060325)}),
			error: nil,
			time:  pointer.To[time.Time](time.Date(2025, 5, 12, 14, 32, 5, 0, time.UTC)),
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			time, err := extractCreationTime(tc.token)
			assert.Equal(t, tc.time, time)
			assert.Equal(t, tc.error, err)
		})
	}
}

func TestIsBlocked(t *testing.T) {
	type tcase struct {
		name      string
		token     *jwt.Token
		isBlocked bool
		error     error
		usersFile string
	}
	tcases := []tcase{
		{
			name:      "token issue date is older than password last edit date",
			isBlocked: true,
			error:     nil,
			token: jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"iat": float64(1747058324), // token creation time is 1 second earlier than the password timestamp
				"sub": "test:login",
				"iss": SessionManagerClaimsIssuer,
			}),
			usersFile: `test:
  enabled: true
  capabilities:
  - login
  passwordMtime: "2025-05-12T18:58:45+05:00"`, // timestamp is 1747058325
		},
		{
			// this case covers:
			// - creating users with the same name as the deleted users
			// - changed password
			name:      "token issue date is younger than password last edit date",
			isBlocked: false,
			error:     nil,
			token: jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"iat": float64(1747058326), // token creation time is 1 second later than the password timestamp
				"sub": "test:login",
				"iss": SessionManagerClaimsIssuer,
			}),
			usersFile: `test:
  enabled: true
  capabilities:
  - login
  passwordMtime: "2025-05-12T18:58:45+05:00"`, // timestamp is 1747058325
		},
		{
			name:      "default admin user without the passwordMtime set",
			isBlocked: false,
			error:     nil,
			token: jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"iat": float64(1847058325),
				"sub": "admin:login",
				"iss": SessionManagerClaimsIssuer,
			}),
			usersFile: `admin:
  enabled: true
  capabilities:
  - login`,
		},
		{
			name: "account not found - block the request",
			token: jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"iat": float64(1847058325),
				"sub": "unknown_account:login",
				"iss": SessionManagerClaimsIssuer,
			}),
			error:     nil,
			isBlocked: true,
			usersFile: `test:
  enabled: true
  capabilities:
  - login
  passwordMtime: "2025-05-12T18:58:45+05:00"`,
		},
		{
			// other error cases are covered by unit tests for specific functions
			name: "error parse passwordMtime",
			token: jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"iat": float64(1847058325),
				"sub": "test:login",
				"iss": SessionManagerClaimsIssuer,
			}), // ts is earlier than the password time
			error:     errors.New(`parsing time "some weird string" as "2006-01-02T15:04:05Z07:00": cannot parse "some weird string" as "2006"`),
			isBlocked: false,
			usersFile: `test:
  enabled: true
  capabilities:
  - login
  passwordMtime: "some weird string"`,
		},
	}
	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			manager, err := mockManager(ctx, tc.usersFile, "")
			assert.NoError(t, err)
			isBlocked, err := manager.IsBlocked(ctx, tc.token)
			if tc.error != nil {
				assert.EqualError(t, err, tc.error.Error())
			}
			assert.Equal(t, tc.isBlocked, isBlocked)
		})
	}
}

func mockManager(ctx context.Context, usersFile, blocklistContent string) (*Manager, error) {
	l := zap.NewNop().Sugar()

	blocklistSecret := getBlockListSecretTemplate(blocklistContent)
	usersSecret := userSecret(usersFile)

	objs := []ctrlclient.Object{blocklistSecret, usersSecret}
	mockClient := fakeclient.NewClientBuilder().WithScheme(kubernetes.CreateScheme())
	mockClient.WithObjects(objs...)

	k := kubernetes.NewEmpty(l).WithKubernetesClient(mockClient.Build())

	bl, err := mockNewBlocklist(ctx, l, k)
	if err != nil {
		return nil, err
	}

	return &Manager{
		accountManager: k.Accounts(),
		signingKey:     nil,
		Blocklist:      bl,
		l:              l,
	}, nil
}

func userSecret(file string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.EverestAccountsSecretName,
			Namespace: common.SystemNamespace,
		},
		Data: map[string][]byte{
			common.EverestAccountsFileName: []byte(file),
		},
	}
}
