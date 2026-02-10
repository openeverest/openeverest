import { Button, Tooltip } from '@mui/material';
import { GenericErrorIcon } from '@percona/ui-lib';
import { NoMatch } from 'pages/404/NoMatch';
import { useContext, useState } from 'react';
import { ErrorContext } from 'utils/ErrorBoundaryProvider';
import { Messages } from './genericError.messages';

export const GenericError = () => {
  const [open, setOpen] = useState(false);
  const { updateError, errorObject, hasError } = useContext(ErrorContext);

  const resetErrorState = () => {
    updateError(false);
  };

  const clipboardAvailable = !!navigator.clipboard;

  const handleCopy = () => {
    if (clipboardAvailable && hasError && errorObject) {
      navigator.clipboard.writeText(
        `Error name: ${errorObject.name}, Error message: ${errorObject.message}, Error stack: ${errorObject.stack}`
      );
      setOpen(true);

      setTimeout(() => {
        setOpen(false);
        window
          .open('https://github.com/openeverest/openeverest/issues', '_blank')
          ?.focus();
      }, 0);
    }
  };

  return (
    <NoMatch
      header={Messages.header}
      subHeader={Messages.subHeader}
      CustomIcon={GenericErrorIcon}
      onButtonClick={resetErrorState}
      customButton={
        <Tooltip
          {...(clipboardAvailable && {
            disableHoverListener: true,
            open,
          })}
          disableFocusListener
          disableTouchListener
          PopperProps={{
            disablePortal: true,
          }}
          title={
            clipboardAvailable ? Messages.copyClipboard : Messages.copyError
          }
        >
          <Button variant="text" onClick={handleCopy}>
            {Messages.copyAndReport}
          </Button>
        </Tooltip>
      }
    />
  );
};
