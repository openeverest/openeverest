import { Authorizer } from 'casbin.js';

let instance: Authorizer;

export const initAuthorizer = async (username: string, policyList: string) => {
  instance = new Authorizer('auto', { endpoint: '/' });
  instance.user = username;
  await instance.initEnforcer(policyList);
};

export const getAuthorizer = () => instance;
