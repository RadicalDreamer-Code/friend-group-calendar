import { EmptyState } from '@/components/EmptyState';
import { Page } from '@/components/Page';

export default function CalendarsScreen() {
  return (
    <Page title="Calendars">
      <EmptyState
        icon="calendar-outline"
        title="No calendars yet"
        message="Create a calendar and invite your friends to plan things together."
      />
    </Page>
  );
}
