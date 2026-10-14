import { StackProps } from '@mui/material';

export type BackNavigationTextProps = {
  text: string;
  onBackClick?: () => void;
  stackProps?: StackProps;
  rightSlot?: React.ReactNode;
};
