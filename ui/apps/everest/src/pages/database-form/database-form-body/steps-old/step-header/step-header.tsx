import { Chip, Typography } from '@mui/material';

type StepHeaderProps = {
  pageTitle?: string;
  pageDescription?: string;
  techPreview?: boolean;
};
export const StepHeader = ({
  pageTitle,
  pageDescription,
  techPreview,
}: StepHeaderProps) => {
  return (
    <>
      {pageTitle && (
        <Typography variant="h5" data-testid="step-header">
          {pageTitle}
          {techPreview && (
            <Chip size="small" label="Technical preview" sx={{ ml: 1 }} />
          )}
        </Typography>
      )}
      {pageDescription && (
        <Typography
          data-testid="step-description"
          variant="subtitle2"
          sx={{ mt: 1, mb: 2 }}
        >
          {pageDescription}
        </Typography>
      )}
    </>
  );
};
