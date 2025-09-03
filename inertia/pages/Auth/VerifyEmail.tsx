import { WarningCircleSolidIcon } from '~/Components/Icons/warning-circle-solid.svg';
import GuestLayout from '~/Layouts/GuestLayout';
import { Head, Link, useForm } from '@inertiajs/react';
import { Button } from '@kibamail/owly';
import * as Alert from '@kibamail/owly/alert';
import { Heading } from '@kibamail/owly/heading';
import { Text } from '@kibamail/owly/text';
import { FormEventHandler } from 'react';

export default function VerifyEmail({ status }: { status?: string }) {
  const { post, processing } = useForm({});

  const submit: FormEventHandler = (e) => {
    e.preventDefault();

    post('/auth/email/verification/send');
  };

  return (
    <GuestLayout>
      <Head title="Email Verification" />

      <div className="mx-auto mt-24 w-full max-w-100">
        <img
          src="/icons/email-send.svg"
          className="mb-4"
          alt="Email sent icon"
        />
        <Heading className="mb-2">Verify your email</Heading>
        <Text className="kb-content-tertiary mt-2">
          Thanks for signing up! Before getting started, could you verify your
          email address by clicking on the link we just emailed to you?
          <span className="mt-2 inline-block">
            If you didn't receive the email, we will gladly send you another.
          </span>
        </Text>

        {status === 'verification-link-sent' && (
          <Alert.Root className="mt-4" variant={'success'}>
            <Alert.Icon>
              <WarningCircleSolidIcon />
            </Alert.Icon>

            <div className="flex w-full flex-col">
              <Alert.Title className="font-medium">Email sent</Alert.Title>

              <Text className="kb-content-secondary mt-1">
                A new verification link has been sent to the email address you
                provided during registration.
              </Text>
            </div>
          </Alert.Root>
        )}

        <form onSubmit={submit}>
          <div className="mt-4 flex w-full flex-col gap-4">
            <Button disabled={processing} width="full">
              Resend Verification Email
            </Button>

            <Button variant="tertiary" asChild width="full">
              <Link href={'/auth/logout'} method="post" as="button">
                Log Out
              </Link>
            </Button>
          </div>
        </form>
      </div>
    </GuestLayout>
  );
}
