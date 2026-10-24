import { router } from 'expo-router';

import { isCalendarGone } from '@/api/calendars';

import { Button } from './Button';
import { EmptyState } from './EmptyState';
import { ErrorState } from './ErrorState';

/** For calendar screens whose calendar could not be loaded. */
export function CalendarError({
  error,
  onRetry,
  retrying,
}: {
  error: unknown;
  onRetry: () => void;
  retrying?: boolean;
}) {
  if (isCalendarGone(error)) {
    return (
      <EmptyState
        icon="calendar-clear-outline"
        title="Calendar not available"
        message="It was deleted, or you are no longer a member."
        action={<Button variant="secondary" title="Back to calendars" onPress={() => router.dismissTo('/')} />}
      />
    );
  }
  return <ErrorState error={error} onRetry={onRetry} retrying={retrying} />;
}
