import { Tooltip, Typography, TypographyProps } from '@mui/material';
import { format, formatDuration, intervalToDuration, isValid } from 'date-fns';
import { DATE_FORMAT } from 'consts';

export type ComponentAgeProps = {
  date: string;
  render?: (date?: string) => React.ReactNode;
  typographyProps?: TypographyProps;
};

const ComponentAge = ({ date, render, typographyProps }: ComponentAgeProps) => {
  const dateObj = new Date(date);

  const formattedDate = isValid(dateObj)
    ? formatDuration(
        intervalToDuration({
          start: date,
          end: Date.now(),
        }),
        {
          format: ['years', 'months', 'days', 'hours', 'minutes'],
        }
      )
    : '';

  const resultStr = formattedDate ? `${formattedDate} ago` : '';

  return (
    <Tooltip
      title={isValid(dateObj) ? `Started at ${format(date, DATE_FORMAT)}` : ''}
      placement="right"
      arrow
    >
      <Typography
        variant="caption"
        {...typographyProps}
        sx={[{
          color: "text.secondary"
        }, ...(typographyProps?.sx
          ? (Array.isArray(typographyProps.sx) ? typographyProps.sx : [typographyProps.sx])
          : [])]}>
        {render ? render(resultStr) : resultStr}
      </Typography>
    </Tooltip>
  );
};

export default ComponentAge;
