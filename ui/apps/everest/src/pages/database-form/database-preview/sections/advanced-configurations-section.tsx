import { PreviewContentText } from '../preview-section';
import { AdvancedConfigurationType } from '../../database-form-schema.ts';
import { ProxyExposeType } from 'shared-types/dbCluster.types';
import { EMPTY_LOAD_BALANCER_CONFIGURATION } from 'consts.ts';

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
}: AdvancedConfigurationType) => {
  const isExternalAccessEnabled =
    exposureMethod === ProxyExposeType.LoadBalancer;

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
