// import InputError from '~/Components/InputError';
// import InputLabel from '~/Components/InputLabel';
// import PrimaryButton from '~/Components/PrimaryButton';
// import TextInput from '~/Components/TextInput';
// import GuestLayout from '~/Layouts/GuestLayout';
// import { Head, useForm } from '@inertiajs/react';
// import { FormEventHandler } from 'react';

// export default function ResetPassword({
//   token,
//   email,
// }: {
//   token: string;
//   email: string;
// }) {
//   const { data, setData, post, processing, errors, reset } = useForm({
//     token: token,
//     email: email,
//     password: '',
//     password_confirmation: '',
//   });

//   const submit: FormEventHandler = (e) => {
//     e.preventDefault();

//     post(route('password.store'), {
//       onFinish: () => reset('password', 'password_confirmation'),
//     });
//   };

//   return (
//     <GuestLayout>
//       <Head title="Reset Password" />

//       <form onSubmit={submit}>
//         <div>
//           <InputLabel htmlFor="email" value="Email" />

//           <TextInput
//             id="email"
//             type="email"
//             name="email"
//             value={data.email}
//             className="mt-1 block w-full"
//             autoComplete="username"
//             onChange={(e) => setData('email', e.target.value)}
//           />

//           <InputError message={errors.email} className="mt-2" />
//         </div>

//         <div className="mt-4">
//           <InputLabel htmlFor="password" value="Password" />

//           <TextInput
//             id="password"
//             type="password"
//             name="password"
//             value={data.password}
//             className="mt-1 block w-full"
//             autoComplete="new-password"
//             isFocused={true}
//             onChange={(e) => setData('password', e.target.value)}
//           />

//           <InputError message={errors.password} className="mt-2" />
//         </div>

//         <div className="mt-4">
//           <InputLabel
//             htmlFor="password_confirmation"
//             value="Confirm Password"
//           />

//           <TextInput
//             type="password"
//             name="password_confirmation"
//             value={data.password_confirmation}
//             className="mt-1 block w-full"
//             autoComplete="new-password"
//             onChange={(e) => setData('password_confirmation', e.target.value)}
//           />

//           <InputError message={errors.password_confirmation} className="mt-2" />
//         </div>

//         <div className="mt-4 flex items-center justify-end">
//           <PrimaryButton className="ms-4" disabled={processing}>
//             Reset Password
//           </PrimaryButton>
//         </div>
//       </form>
//     </GuestLayout>
//   );
// }

import { PasswordField } from '~/Components/password-field.jsx';
import GuestLayout from '~/Layouts/GuestLayout';

import { Head, useForm } from '@inertiajs/react';
import { Button } from '@kibamail/owly/button';
import { Heading } from '@kibamail/owly/heading';
import { Text } from '@kibamail/owly/text';
import * as TextField from '@kibamail/owly/text-field';
import { FormEventHandler } from 'react';

export default function ResetPasswordPage({
  token,
  email,
}: {
  token: string;
  email: string;
}) {
  const { data, setData, post, processing, errors, reset } = useForm({
    token: token,
    email: email,
    password: '',
    password_confirmation: '',
  });

  const submit: FormEventHandler = (e) => {
    e.preventDefault();

    post(route('password.store'), {
      onFinish: () => reset('password', 'password_confirmation'),
    });
  };

  const isSuccess = false;

  if (isSuccess) {
    return (
      <div className="mx-auto mt-24 w-full max-w-100">
        <img
          src="/icons/email-send.svg"
          className="mb-4"
          alt="Password reset success icon"
        />
        <Heading>Your password was updated successfully.</Heading>

        <Text className="kb-content-tertiary mt-2">
          You're all set. You may login with your new password now.
        </Text>

        <Button className="mt-10" width={'full'} variant="secondary" asChild>
          <a href={route('auth_login')}>Back to login</a>
        </Button>
      </div>
    );
  }

  return (
    <GuestLayout>
      <Head title="Reset Password" />
      <div className="mx-auto mt-24 w-full max-w-100">
        <Heading>Create a new password</Heading>

        <Text className="kb-content-tertiary mt-2">
          Create a new password to regain access to your account
        </Text>

        <form className="mt-10 flex flex-col" onSubmit={submit}>
          <div className="grid grid-cols-1 gap-4">
            <TextField.Root
              id="email"
              type="email"
              name="email"
              required
              value={data.email}
              onChange={(e) => setData('email', e.target.value)}
              placeholder="Enter your account email address"
            >
              <TextField.Label htmlFor="email">Email address</TextField.Label>

              {errors?.email ? (
                <TextField.Error>{errors?.email}</TextField.Error>
              ) : null}
            </TextField.Root>

            <div className="relative">
              <PasswordField
                strengthIndicator
                id="new-password"
                name="password"
                required
                value={data.password}
                onChange={(e) => setData('password', e.target.value)}
                placeholder="Choose a new password"
              >
                <TextField.Label htmlFor="new-password">
                  New password
                </TextField.Label>

                {errors?.password ? (
                  <TextField.Error className="mt-6">
                    {errors?.password}
                  </TextField.Error>
                ) : null}
              </PasswordField>
            </div>

            <PasswordField
              placeholder="Confirm your password"
              id="confirm-password"
              name="password_confirmation"
              required
              value={data.password_confirmation}
              onChange={(e) => setData('password_confirmation', e.target.value)}
            >
              <TextField.Label htmlFor="confirm-password">
                Confirm password
              </TextField.Label>
              {errors?.password_confirmation ? (
                <TextField.Error>
                  {errors?.password_confirmation}
                </TextField.Error>
              ) : null}
            </PasswordField>
          </div>

          <div className="mt-6 grid w-full grid-cols-1 gap-2">
            <Button loading={processing} type="submit" width={'full'}>
              Continue
            </Button>
          </div>
        </form>
      </div>
    </GuestLayout>
  );
}
