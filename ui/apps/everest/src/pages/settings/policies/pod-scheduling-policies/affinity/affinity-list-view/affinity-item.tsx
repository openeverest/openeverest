import { Stack, Typography } from '@mui/material';
import { AffinityRule, AffinityType } from 'shared-types/affinity.types';

const showRuleProperty = (prop: string | undefined) => {
  return prop ? ` | ${prop}` : '';
};

export const AffinityItem = ({ rule }: { rule: AffinityRule }) => {
  const valuesToShow =
    rule.type === AffinityType.NodeAffinity ? [] : [rule.topologyKey];
  return (
    <Stack
      direction="row"
      sx={{
        alignItems: "center",
        width: '100%'
      }}>
      <Stack
        sx={{
          width: '50%',
        }}
      >
        <Typography variant="body1">
          {rule.type}
          {[...valuesToShow, rule.key, rule.operator, rule.values].map((prop) =>
            showRuleProperty(prop)
          )}
        </Typography>
      </Stack>
    </Stack>
  );
};
