import { AccordionSummary, Box, Typography } from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';

const CustomAccordionSummary = ({
  unitPlural,
  nr,
  hasError,
  title,
}: {
  unitPlural?: string;
  nr?: number;
  hasError?: boolean;
  title?: string;
}) => {
  const text = Number.isNaN(nr ?? NaN) || (nr ?? 0) < 1 ? '' : ` (${nr})`;

  return (
    <AccordionSummary
      sx={{
        paddingLeft: 0,
      }}
      expandIcon={<ExpandMoreIcon />}
    >
      <Box display="flex" alignItems="center">
        {hasError && (
          <ErrorOutlineIcon
            color="error"
            sx={{ mr: 1, position: 'relative', bottom: 1 }}
          />
        )}
        {unitPlural && (
          <Typography
            variant="sectionHeading"
            textTransform="capitalize"
          >{`${unitPlural} ${text}`}</Typography>
        )}

        {title && (
          <Typography variant="h6" sx={{ pl: 3, pt: 1 }}>
            {title}
          </Typography>
        )}
      </Box>
    </AccordionSummary>
  );
};

export default CustomAccordionSummary;
