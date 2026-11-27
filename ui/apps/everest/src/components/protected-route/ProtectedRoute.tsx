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

import { useAuth } from 'oidc-react';
import { ReactNode, useContext, useLayoutEffect, useRef } from 'react';
import { Navigate, Outlet, useLocation } from 'react-router-dom';
import LoadingPageSkeleton from 'components/loading-page-skeleton/LoadingPageSkeleton';
import { AuthContext } from 'contexts/auth';

const ProtectedRoute = ({ children }: { children?: ReactNode }) => {
  const location = useLocation();
  const {
    authStatus,
    setRedirectRoute,
    isSsoEnabled,
    isLogoutIntentional = false,
  } = useContext(AuthContext);
  const { isLoading } = useAuth();
  const isInitialVisit = useRef(true);

  // A fresh protected visit may set a target after logout; the page being logged out must not.
  // Consume the target before the protected page can navigate again.
  useLayoutEffect(() => {
    if (
      authStatus === 'unknown' ||
      (authStatus === 'loggedOut' &&
        (!isLogoutIntentional || isInitialVisit.current))
    ) {
      setRedirectRoute(location.pathname + location.search + location.hash);
    } else if (authStatus === 'loggedIn') {
      setRedirectRoute(null);
    }
    isInitialVisit.current = false;
  }, [
    authStatus,
    location.pathname,
    location.search,
    location.hash,
    setRedirectRoute,
    isLogoutIntentional,
  ]);

  // At this point, we're pretty much checking the auth state.
  // Later this can be some sort of loading UI
  if (
    authStatus === 'unknown' ||
    authStatus === 'loggingIn' ||
    authStatus === 'loggingOut' ||
    (isSsoEnabled && isLoading)
  ) {
    return <LoadingPageSkeleton />;
  }

  if (authStatus === 'loggedOut') {
    return <Navigate to="/login" replace />;
  }

  return children ? children : <Outlet />;
};

export default ProtectedRoute;
