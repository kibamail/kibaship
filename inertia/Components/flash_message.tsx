// import "./flash_message.css";
import { WarningCircleSolidIcon } from "~/Components/Icons/warning-circle-solid.svg";
import * as Alert from "@kibamail/owly/alert";
import { Text } from "@kibamail/owly/text";

interface FlashMessageProps extends Alert.AlertRootProps {
    alert?: {
        title?: string;
        description?: string;
        variant?: Alert.AlertRootProps["variant"];
    };
}

export function FlashMessage({ alert, ...rootProps }: FlashMessageProps) {
    return (
        <Alert.Root {...rootProps} variant={alert?.variant}>
            <Alert.Icon>
                <WarningCircleSolidIcon />
            </Alert.Icon>

            <div className="w-full flex flex-col">
                <Alert.Title className="font-medium">
                    {alert?.title}
                </Alert.Title>

                {alert?.description ? (
                    <Text className="kb-content-secondary mt-1">
                        {alert?.description}
                    </Text>
                ) : null}
            </div>
        </Alert.Root>
    );
}
