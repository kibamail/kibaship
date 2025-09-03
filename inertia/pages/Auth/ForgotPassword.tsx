import { Head, Link, useForm } from '@inertiajs/react';
import { FormEventHandler } from 'react';

// export default function ForgotPassword({ status }: { status?: string }) {
//     const { data, setData, post, processing, errors } = useForm({
//         email: '',
//     });

//     const submit: FormEventHandler = (e) => {
//         e.preventDefault();

//         post(route('password.email'));
//     };

//     return (
//         <GuestLayout>
//             <Head title="Forgot Password" />

//             <div className="mb-4 text-sm text-gray-600 dark:text-gray-400">
//                 Forgot your password? No problem. Just let us know your email
//                 address and we will email you a password reset link that will
//                 allow you to choose a new one.
//             </div>

//             {status && (
//                 <div className="mb-4 text-sm font-medium text-green-600 dark:text-green-400">
//                     {status}
//                 </div>
//             )}

//             <form onSubmit={submit}>
//                 <TextInput
//                     id="email"
//                     type="email"
//                     name="email"
//                     value={data.email}
//                     className="mt-1 block w-full"
//                     isFocused={true}
//                     onChange={(e) => setData('email', e.target.value)}
//                 />

//                 <InputError message={errors.email} className="mt-2" />

//                 <div className="mt-4 flex items-center justify-end">
//                     <PrimaryButton className="ms-4" disabled={processing}>
//                         Email Password Reset Link
//                     </PrimaryButton>
//                 </div>
//             </form>
//         </GuestLayout>
//     );
// }

import GuestLayout from '~/Layouts/GuestLayout';
import { Button } from '@kibamail/owly/button';
import { Heading } from '@kibamail/owly/heading';
import { Text } from '@kibamail/owly/text';
import * as TextField from '@kibamail/owly/text-field';

export default function ForgotPassword({
  status,
  success,
}: {
  status?: string;
  success?: string;
}) {
  const { data, setData, post, processing, errors } = useForm({
    email: '',
  });

  const submit: FormEventHandler = (e) => {
    e.preventDefault();

    post(route('password.email'));
  };

  const isSuccess = success === 'true';

  return (
    <GuestLayout>
      <Head title="Forgot Password" />
      <div className="mx-auto mt-24 w-full max-w-100">
        {isSuccess ? (
          <img
            src="/icons/email-send.svg"
            className="mb-4"
            alt="Email sent icon"
          />
        ) : null}
        <Heading className="mb-2">Reset password</Heading>

        {isSuccess ? (
          <Text className="kb-content-tertiary mt-2">
            We received your request to reset your email. If an account exists
            with the email you entered. You'll receive an email with a password
            reset link soon.
          </Text>
        ) : (
          <Text className="kb-content-tertiary mt-2">
            Enter your email address. If an account exists, you{"'"}ll receive
            an email with a password reset link soon.
          </Text>
        )}

        {isSuccess ? (
          <Button className="mt-10" width={'full'} variant="secondary" asChild>
            <Link href={route('login')}>Back to login</Link>
          </Button>
        ) : null}

        {isSuccess ? null : (
          <form onSubmit={submit} className="mt-10 flex flex-col">
            <TextField.Root
              required
              id="email"
              type="email"
              name="email"
              value={data.email}
              onChange={(e) => setData('email', e.target.value)}
              placeholder="Enter your account email address"
            >
              <TextField.Label htmlFor="email">Email address</TextField.Label>

              {errors?.email ? (
                <TextField.Error> {errors?.email} </TextField.Error>
              ) : null}
            </TextField.Root>

            <div className="mt-6 grid w-full grid-cols-1 gap-2">
              <Button type="submit" width={'full'} loading={processing}>
                Continue
              </Button>

              <Button variant="tertiary" width="full" asChild>
                <Link href={route('login')}>Back to login</Link>
              </Button>
            </div>
          </form>
        )}
      </div>
    </GuestLayout>
  );
}
