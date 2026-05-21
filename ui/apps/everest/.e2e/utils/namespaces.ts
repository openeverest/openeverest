import { APIRequestContext, expect, Page } from '@playwright/test';
import { EVEREST_CI_NAMESPACES } from '../constants';
export const getNamespacesFn = async (
  token: string,
  request: APIRequestContext,
  clusterName = 'main'
) => {
  const namespacesInfo = await request.get(
    `/v1/clusters/${clusterName}/namespaces`,
    {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    }
  );
  expect(namespacesInfo.ok()).toBeTruthy();
  return namespacesInfo.json();
};

export const setNamespace = async (
  page: Page,
  namespaceName: EVEREST_CI_NAMESPACES
) => {
  await page.getByTestId('k8s-namespace-autocomplete').click();
  await page.getByRole('option', { name: namespaceName }).click();
  await expect(page.getByTestId('text-input-k8s-namespace')).not.toBeEmpty();
};
