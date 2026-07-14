// Copyright (C) 2026 The OpenEverest Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import LoadingPageSkeleton from 'components/loading-page-skeleton/LoadingPageSkeleton';

// This is the component rendered at the OIDC redirect route (/login-callback).
// It is only a loading placeholder: the authorization-code response is processed
// by oidc-react's AuthProvider (see App.tsx `onSignIn`/`onSignInError`), which is
// the single owner of the OIDC state and performs the redirect once done.
const LoginCallbackLoader = () => <LoadingPageSkeleton />;

export default LoginCallbackLoader;
