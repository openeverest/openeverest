// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/pkg/kubernetes"
)

func TestShouldAllowRequestDuringEngineUpgrade(t *testing.T) {
	lockedAt := time.Now().Format(time.RFC3339)
	t.Parallel()
	testCases := []struct {
		description string
		objs        []ctrlclient.Object
		ctxFn       func() echo.Context
		allow       bool
	}{
		{
			description: "allow all GET requests",
			ctxFn: func() echo.Context {
				return echo.New().NewContext(&http.Request{
					Method: http.MethodGet,
				}, nil,
				)
			},
			allow: true,
		},
		{
			description: "allow non-target paths",
			ctxFn: func() echo.Context {
				return echo.New().NewContext(&http.Request{
					Method: http.MethodPost,
					URL: &url.URL{
						Path: "/api/v1/namespaces/default/monitoring-instances",
					},
				}, nil,
				)
			},
			allow: true,
		},
		{
			description: "allow target paths with no namespace",
			ctxFn: func() echo.Context {
				return echo.New().NewContext(&http.Request{
					Method: http.MethodPost,
					URL: &url.URL{
						Path: "/api/v1/database-clusters",
					},
				}, nil,
				)
			},
			allow: true,
		},
		{
			description: "allow target path with no lock annotation",
			ctxFn: func() echo.Context {
				ctx := echo.New().NewContext(&http.Request{
					Method: http.MethodDelete,
					URL: &url.URL{
						Path: "/api/v1/namespaces/default/database-clusters/1234",
					},
				}, nil,
				)
				ctx.SetParamNames("namespace")
				ctx.SetParamValues("default")
				return ctx
			},
			objs: []ctrlclient.Object{
				&everestv1alpha1.DatabaseEngine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-engine",
						Namespace: "default",
					},
				},
			},
			allow: true,
		},
		{
			description: "deny request on target path with lock annotation",
			ctxFn: func() echo.Context {
				ctx := echo.New().NewContext(&http.Request{
					Method: http.MethodDelete,
					URL: &url.URL{
						Path: "/api/v1/namespaces/default/database-clusters/1234",
					},
				}, nil,
				)
				ctx.SetParamNames("namespace")
				ctx.SetParamValues("default")
				return ctx
			},
			objs: []ctrlclient.Object{
				&everestv1alpha1.DatabaseEngine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-engine",
						Namespace: "default",
						Annotations: map[string]string{
							everestv1alpha1.DatabaseOperatorUpgradeLockAnnotation: lockedAt,
						},
					},
				},
			},
			allow: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			t.Parallel()
			mockClient := fakeclient.NewClientBuilder().WithScheme(kubernetes.CreateScheme()).WithObjects(tc.objs...)
			k := kubernetes.NewEmpty(zap.NewNop().Sugar()).WithKubernetesClient(mockClient.Build())
			e := EverestServer{kubeConnector: k}
			ctx := tc.ctxFn()

			allow, err := e.shouldAllowRequestDuringEngineUpgrade(ctx)
			require.NoError(t, err)
			assert.Equal(t, tc.allow, allow)
		})
	}
}
