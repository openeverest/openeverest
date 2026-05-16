// everest
// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
import CronTime from 'cron-time-generator';
import {
  TimeProps,
  TimeValue,
  timeValueHumanized,
} from 'components/time-selection/time-selection.types';
import { addZeroToSingleDigit } from 'components/time-selection/time-selection.utils';

export const Messages = {
  title: 'Database Summary',
  showMore: 'Show more',
  showLess: 'Show less',
};

export const getTimeSelectionPreviewMessage = ({
  selectedTime,
  minute,
  hour,
  amPm,
  onDay,
  weekDay,
}: TimeProps) => {
  const minuteWithZero = addZeroToSingleDigit(minute);

  switch (selectedTime) {
    case TimeValue.hours:
      return `Every hour at minute ${minute}`;
    case TimeValue.days:
      return `${timeValueHumanized[selectedTime]} at ${hour}:${minuteWithZero} ${amPm}`;
    case TimeValue.weeks:
      return `${timeValueHumanized[selectedTime]} on ${weekDay}s at ${hour}:${minuteWithZero} ${amPm}`;
    case TimeValue.months:
      return `${timeValueHumanized[selectedTime]} on day ${onDay} at ${hour}:${minuteWithZero} ${amPm}`;
    default:
      return CronTime.everyHourAt(5);
  }
};
