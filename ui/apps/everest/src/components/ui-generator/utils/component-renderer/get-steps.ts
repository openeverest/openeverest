import type {
  Section,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';

export const getSteps = (
  selectedTopology: string,
  topologyUiSchemas: TopologyUISchemas
): { [key: string]: Section } => {
  return topologyUiSchemas[selectedTopology]?.sections || {};
};
