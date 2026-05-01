import { PreviewContentText } from '../preview-section';
import { AdvancedConfigurationType } from '../../database-form-schema.ts';
import { ProxyExposeType } from 'shared-types/dbCluster.types';
import { EMPTY_LOAD_BALANCER_CONFIGURATION } from 'consts.ts';
import { DbType } from '@percona/types';
import { getProxyConfigLabel } from 'components/cluster-form/advanced-configuration/advanced-configuration.utils';

type AdvancedConfigurationsPreviewProps = AdvancedConfigurationType & {
  dbType?: DbType;
  sharding?: boolean;
};

export const AdvancedConfigurationsPreviewSection = ({
  exposureMethod,
  loadBalancerConfigName,
  engineParametersEnabled,
  engineParameters,
  storageClass,
  podSchedulingPolicyEnabled,
  podSchedulingPolicy,
  splitHorizonDNSEnabled,
  splitHorizonDNS,
  proxyConfigEnabled,
  proxyConfig,
  dbType,
  sharding,
}: AdvancedConfigurationsPreviewProps) => {
  const isExternalAccessEnabled =
    exposureMethod === ProxyExposeType.LoadBalancer;
  const showProxyConfig =
    dbType !== undefined ? !(dbType === DbType.Mongo && !sharding) : true;

  return (
    <>
      <PreviewContentText text={`Storage class: ${storageClass ?? ''}`} />
      <PreviewContentText
        text={`Ext. access: ${isExternalAccessEnabled ? 'enabled' : 'disabled'}`}
      />
      {isExternalAccessEnabled && (
        <PreviewContentText
          text={`Config name: ${loadBalancerConfigName ?? EMPTY_LOAD_BALANCER_CONFIGURATION}`}
        />
      )}
      {engineParametersEnabled && engineParameters && (
        <PreviewContentText text="Database engine parameters set" />
      )}
      {showProxyConfig && proxyConfigEnabled && proxyConfig && dbType && (
        <PreviewContentText text={`${getProxyConfigLabel(dbType)} set`} />
      )}
      {podSchedulingPolicyEnabled && podSchedulingPolicy && (
        <PreviewContentText
          text={`Pod scheduling policy: ${podSchedulingPolicy}`}
        />
      )}
      {splitHorizonDNSEnabled && splitHorizonDNS && (
        <PreviewContentText text={`Split-horizon DNS: ${splitHorizonDNS}`} />
      )}
    </>
  );
};
