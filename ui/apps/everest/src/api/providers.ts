import { api } from './api';
import { ProviderList } from 'types/api';


export const getProvidersFn = async () => {
  const response = await api.get<ProviderList>(
    `/clusters/main/providers`
  );
  return response.data;
}
