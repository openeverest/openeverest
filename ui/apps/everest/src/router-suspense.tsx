import { Suspense, type ReactNode } from 'react';

export const withSuspense = (element: ReactNode) => (
  <Suspense fallback={<div>Loading...</div>}>{element}</Suspense>
);