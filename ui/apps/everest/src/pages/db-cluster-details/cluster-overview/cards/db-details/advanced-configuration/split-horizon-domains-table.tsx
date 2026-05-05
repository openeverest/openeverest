import { Table } from '@percona/ui-lib';
import { useMemo } from 'react';

const SplitHorizonDomainsTable = ({
  domains,
}: {
  domains: { domain?: string; privateIP?: string; publicIP?: string }[];
}) => {
  const columns = useMemo(
    () => [
      {
        header: 'Domain',
        accessorKey: 'domain',
        minSize: 300,
      },
      {
        header: 'Private IP',
        accessorKey: 'privateIP',
      },
      {
        header: 'Public IP',
        accessorKey: 'publicIP',
      },
    ],
    []
  );

  return (
    <Table tableName="split-horizon-domains" columns={columns} data={domains} />
  );
};

export default SplitHorizonDomainsTable;
