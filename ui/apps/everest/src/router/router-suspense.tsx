import LoadingPageSkeleton from 'components/loading-page-skeleton/LoadingPageSkeleton';
import { Suspense, type ReactNode } from 'react';

export const withSuspense = (element: ReactNode) => (
  <Suspense fallback={<LoadingPageSkeleton />}>{element}</Suspense>
);
