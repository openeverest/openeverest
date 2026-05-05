export type SplitHorizonDNSConfig = {
  apiVersion: 'everest.percona.com/v1alpha1';
  kind: 'SplitHorizonDNSConfig';
  metadata: {
    name: string;
  };
  spec: {
    baseDomainNameSuffix: string;
    tls: {
      secretName: string;
      certificate: {
        'ca.crt': string;
        'ca.key': string;
      };
    };
  };
  status: {
    inUse: boolean;
  };
};
