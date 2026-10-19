import { EmptyState } from '@/components/EmptyState';
import { Page } from '@/components/Page';

export default function FriendsScreen() {
  return (
    <Page title="Friends">
      <EmptyState
        icon="people-outline"
        title="No friends yet"
        message="Add friends by their username. Their requests will show up here too."
      />
    </Page>
  );
}
