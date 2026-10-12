import { AxiosAdapter, AxiosError, AxiosResponse } from 'axios';
import {
  addApiAuthInterceptor,
  addApiErrorInterceptor,
  api,
  setTokenRefresher,
} from './api';

const navigations: string[] = [];
let acceptedToken = 'renewed';

const fakeServer: AxiosAdapter = async (config) => {
  const accepted =
    config.headers['Authorization'] === `Bearer ${acceptedToken}`;
  const response: AxiosResponse = {
    data: accepted ? {} : { message: 'invalid or expired jwt' },
    status: accepted ? 200 : 401,
    statusText: '',
    headers: {},
    config,
  };
  if (accepted) {
    return response;
  }
  throw new AxiosError(
    'Rejected',
    AxiosError.ERR_BAD_REQUEST,
    config,
    {},
    response
  );
};

const renewingRefresher = () =>
  vi.fn(
    () =>
      new Promise<string>((resolve) =>
        setTimeout(() => {
          localStorage.setItem('everestToken', 'renewed');
          resolve('renewed');
        })
      )
  );

describe('api 401 handling', () => {
  beforeEach(() => {
    acceptedToken = 'renewed';
    navigations.length = 0;
    localStorage.setItem('everestToken', 'stale');
    addApiErrorInterceptor();
    addApiAuthInterceptor();
    api.defaults.adapter = fakeServer;
    vi.stubGlobal('location', {
      set href(url: string) {
        navigations.push(url);
      },
    });
  });

  afterEach(() => {
    setTokenRefresher(null);
    vi.unstubAllGlobals();
  });

  it('runs one renewal for concurrent 401s and passes the rejected token', async () => {
    const refresher = renewingRefresher();
    setTokenRefresher(refresher);

    await Promise.all([api.get('/a'), api.get('/b')]);

    expect(refresher).toHaveBeenCalledTimes(1);
    expect(refresher).toHaveBeenCalledWith('stale');
    expect(navigations).toEqual([]);
  });

  it('logs out instead of renewing again when the replayed request is rejected', async () => {
    acceptedToken = 'never';
    const refresher = renewingRefresher();
    setTokenRefresher(refresher);

    await api.get('/a');

    expect(refresher).toHaveBeenCalledTimes(1);
    expect(navigations).toEqual(['/logout']);
  });

  it('never renews requests marked with skipAuthRefresh', async () => {
    const refresher = renewingRefresher();
    setTokenRefresher(refresher);

    await api.post('/session/sso', {}, { skipAuthRefresh: true });

    expect(refresher).not.toHaveBeenCalled();
    expect(navigations).toEqual(['/logout']);
  });
});
