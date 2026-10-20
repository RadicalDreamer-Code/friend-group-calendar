import { router } from 'expo-router';
import { useState } from 'react';

import { useRequestCode } from '@/api/account';
import { ApiError, errorMessage } from '@/api/client';
import { useSignInFlow } from '@/auth/signInFlow';
import { AuthScreen } from '@/components/AuthScreen';
import { Button } from '@/components/Button';
import { TextField } from '@/components/TextField';

function isRateLimited(e: unknown) {
  return e instanceof ApiError && e.code === 'rate_limited';
}

export default function SignInScreen() {
  const flow = useSignInFlow();
  const [email, setEmail] = useState(flow.email ?? '');
  const requestCode = useRequestCode();

  const submit = () => {
    const address = email.trim();
    if (!address || requestCode.isPending) return;
    requestCode.mutate(address, {
      onSuccess: (challengeId) => {
        flow.setChallenge(address, challengeId);
        router.push('/verify');
      },
      // Waiting out the resend cooldown is no reason to block someone who
      // already has a code for this address in their inbox.
      onError: (e) => {
        if (isRateLimited(e) && hasCodeFor(address)) router.push('/verify');
      },
    });
  };

  const hasCodeFor = (address: string) =>
    flow.challengeId !== null && flow.email?.toLowerCase() === address.toLowerCase();

  const error = requestCode.error && !(isRateLimited(requestCode.error) && hasCodeFor(email.trim()));

  return (
    <AuthScreen
      title="Sign in"
      subtitle="Enter your email and we’ll send you a code. New here? This creates your account."
    >
      <TextField
        label="Email"
        value={email}
        onChangeText={(text) => {
          setEmail(text);
          if (requestCode.isError) requestCode.reset();
        }}
        onSubmitEditing={submit}
        placeholder="you@example.com"
        autoFocus
        autoCapitalize="none"
        autoCorrect={false}
        autoComplete="email"
        inputMode="email"
        keyboardType="email-address"
        textContentType="emailAddress"
        returnKeyType="send"
        error={error ? errorMessage(requestCode.error) : null}
      />
      <Button title="Continue" onPress={submit} loading={requestCode.isPending} disabled={!email.trim()} />
    </AuthScreen>
  );
}
