import { Text } from '@kibamail/owly/text'

/**
 * DockerImageConfig Component
 * 
 * Configuration component for Docker image application type.
 * This component will handle Docker-specific settings like:
 * - Image registry and name
 * - Tag/version selection
 * - Environment variables
 * - Port configuration
 * - Volume mounts
 * - Resource limits
 */
export function DockerImageConfig() {
  return (
    <div className="space-y-4">
      <div className="p-4 bg-owly-background-secondary rounded-lg border border-owly-border-tertiary">
        <Text className="text-sm text-owly-content-secondary">
          Docker image configuration will be implemented here.
        </Text>
        <Text className="text-xs text-owly-content-tertiary mt-1">
          This will include image selection, environment variables, ports, volumes, and resource limits.
        </Text>
      </div>
    </div>
  )
}
