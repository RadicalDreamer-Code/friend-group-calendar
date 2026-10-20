import { createContext, use, useCallback, useMemo, useState, type ReactNode } from 'react';

/**
 * Carries the email and the code's challenge ID from the first sign-in step
 * to the code step. They stay in memory rather than in the URL so the address
 * does not end up in browser history; reloading the code step starts over.
 */
type SignInFlow = {
  email: string | null;
  /** From the last code requested for `email`; verify needs it with the code. */
  challengeId: string | null;
  setChallenge: (email: string, challengeId: string) => void;
};

const SignInFlowContext = createContext<SignInFlow | null>(null);

export function SignInFlowProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<{ email: string; challengeId: string } | null>(null);
  const setChallenge = useCallback((email: string, challengeId: string) => setState({ email, challengeId }), []);
  const value = useMemo(
    () => ({ email: state?.email ?? null, challengeId: state?.challengeId ?? null, setChallenge }),
    [state, setChallenge],
  );
  return <SignInFlowContext value={value}>{children}</SignInFlowContext>;
}

export function useSignInFlow(): SignInFlow {
  const ctx = use(SignInFlowContext);
  if (!ctx) throw new Error('useSignInFlow must be used inside <SignInFlowProvider>');
  return ctx;
}
