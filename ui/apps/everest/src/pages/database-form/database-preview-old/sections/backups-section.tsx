//@ts-nocheck
// TODO remove this file after release of v2

import { PreviewContentText } from '../../database-preview/preview-section.tsx';
import { getTimeSelectionPreviewMessage } from '../../database-preview/database.preview.messages.ts';
import { BackupStepType } from '../../database-form-schema.ts';
import { getFormValuesFromCronExpression } from 'components/time-selection/time-selection.utils.ts';

export const BackupsPreviewSection = (backupsSection: BackupStepType) => {
  const { schedules, pitrEnabled } = backupsSection;
  const schedulesPreviewList = schedules.map((item) =>
    getTimeSelectionPreviewMessage(
      getFormValuesFromCronExpression(item.schedule)
    )
  );

  return (
    <>
      {schedulesPreviewList.length > 0 ? (
        schedulesPreviewList.map((item) => (
          <PreviewContentText key={`${item}`} text={item} />
        ))
      ) : (
        <PreviewContentText
          text="Backups disabled"
          dataTestId="empty-backups"
        />
      )}
      <PreviewContentText
        text={pitrEnabled ? 'PITR Enabled' : 'PITR Disabled'}
      />
    </>
  );
};
