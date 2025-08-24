import vine from '@vinejs/vine'
import { FieldContext, Validator } from '@vinejs/vine/types'
import { CloudProviderDefinitions } from '#services/cloud-providers/cloud_provider_definitions'
import { SshKeyService } from '#services/ssh/ssh_key_service'
import { HetznerService } from '#services/hetzner/hetzner_service'

const hetznerProviderValidator = async (
    credentials: Record<string, string>,
    field: FieldContext
) => {
    const { publicKey, id } = await SshKeyService.generateEd25519KeyPair()

    if (!publicKey) {
        field.report(
            'Failed to perform credentials validation. Please try again.',
            'credentials.token',
            field
        )

        return
    }

    const SSH_KEYPAIR_NAME = `kibaship-${id}`

    const [sshKeypair, error] = await new HetznerService(credentials.token)
        .sshkeys()
        .create(SSH_KEYPAIR_NAME, publicKey)

    if (error) {
        field.report('Invalid Hetzner token', 'credentials.token', field)

        return
    }

    await new HetznerService(credentials.token).sshkeys().delete(sshKeypair.data.ssh_key.id)
}

const credentialsValidator: Validator<Record<string, string>> = async (
    value: unknown,
    _options: Record<string, string>,
    field: FieldContext
) => {
    if (typeof value !== 'object') {
        return
    }

    if (!field.data.type) {
        return
    }

    if (field.data.type === CloudProviderDefinitions.HETZNER) {
        return hetznerProviderValidator(value as Record<string, string>, field)
    }
}

const credentialsValidatorRule = vine.createRule(credentialsValidator)

export const createCloudProviderValidator = vine.compile(
    vine.object({
        name: vine.string().trim().minLength(1).maxLength(16),
        type: vine.enum(CloudProviderDefinitions.implemented()),
        credentials: vine
            .object({
                token: vine
                    .string()
                    .optional()
                    .requiredWhen('type', 'in', [
                        CloudProviderDefinitions.HETZNER,
                        CloudProviderDefinitions.DIGITAL_OCEAN,
                    ]),
            })
            .use(credentialsValidatorRule({})),
    })
)
