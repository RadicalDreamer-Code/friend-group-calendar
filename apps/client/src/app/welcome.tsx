import { useSignOut } from '@/api/account';
import { AuthScreen } from '@/components/AuthScreen';
import { Button } from '@/components/Button';
import { UsernameForm } from '@/components/UsernameForm';

/** Shown once, right after the first sign-in, until a username is picked. */
export default function WelcomeScreen() {
  const signOut = useSignOut();
  return (
    <AuthScreen
      title="Pick a username"
      subtitle="Friends find and invite you by your username. You can change it later."
    >
      {/* Saving updates the session, which moves on to the app by itself. */}
      <UsernameForm submitLabel="Continue" />
      <Button variant="plain" title="Sign out" onPress={() => signOut.mutate()} />
    </AuthScreen>
  );
}
