import { useEffect } from 'react';
import { useAuth } from 'oidc-react';
import { api } from 'api/api';

const LoginCallback = () => {
  const { userManager } = useAuth();
  useEffect(() => {
    const processLogin = async () => {
      try {
        const user = await userManager.signinCallback();

        if (user) {
          // Exchange the OIDC access token for an Everest JWT via the SSO token exchange endpoint.
          // This supports both JWT and opaque access tokens (e.g. Authentik).
          const response = await api.post('/session/sso', {
            token: user.access_token,
          });
          localStorage.setItem('everestToken', response.data.token);
          window.location.href = '/';
        }
      } catch (error) {
        return;
      }
    };
    processLogin();
  }, [userManager]);

  return null;
};

export default LoginCallback;
