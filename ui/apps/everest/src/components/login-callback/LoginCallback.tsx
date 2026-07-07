import LoadingPageSkeleton from 'components/loading-page-skeleton/LoadingPageSkeleton';

// The OIDC authorization-code response is processed by oidc-react's
// AuthProvider (see App.tsx `onSignIn`/`onSignInError`), which is the single
// owner of the OIDC state. Calling userManager.signinCallback() here as well
// used to race with it for the same stored state and fail with
// "No matching state found in storage", leaving the user on a blank page.
// This route only shows a loader until that handler redirects.
const LoginCallback = () => <LoadingPageSkeleton />;

export default LoginCallback;
