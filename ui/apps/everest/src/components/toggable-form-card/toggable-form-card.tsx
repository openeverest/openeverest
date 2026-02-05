import { SwitchProps, Tooltip } from '@mui/material';
import { SwitchInput } from '@percona/ui-lib';
import { FormCard } from 'components/form-card';

const ToggableFormCard = ({
  title,
  description,
  bottomSlot,
  switchInputName,
  tooltipText,
  switchFieldProps,
}: {
  title: string;
  description: string;
  bottomSlot: React.ReactNode;
  switchInputName: string;
  tooltipText?: string;
  switchFieldProps?: SwitchProps;
}) => {
  return (
    <FormCard
      title={title}
      description={description}
      cardContent={bottomSlot}
      controlComponent={
        <Tooltip title={tooltipText} placement="top" arrow>
          <span>
            <SwitchInput
              label="Enable"
              name={switchInputName}
              switchFieldProps={switchFieldProps}
            />
          </span>
        </Tooltip>
      }
    />
  );
};

export default ToggableFormCard;
