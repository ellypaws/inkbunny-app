import React from 'react';
import {
    ChatItem,
    ChatItemProps,
    Highlighter,
    StoryBook,
    useControls,
    useCreateStore,
} from '@lobehub/ui';

import { avatar } from './data';

interface FunctionProps {
    value?: string;
    useLevaStore?: boolean;
}

const demoError = {
    details: {
        exception:
            'Validation filter failedId-f5aab7304f6c754804f70000Id-f5aab7304f6c754804f70000Id-f5aab7304f6c754804f70000Id-f5aab7304f6c754804f70000Id-f5aab7304f6c754804f70000Id-f5aab7304f6c754804f70000',
        msgId:
            'Id-f5aab7304f6c754804f70000Id-f5aab7304f6c754804f70000Id-f5aab7304f6c754804f70000Id-f5aab7304f6c754804f70000Id-f5aab7304f6c754804f70000',
    },
    reasons: [
        {
            language: 'en',
            message: 'Validation filter failed',
        },
    ],
};

export const Chat: React.FC<FunctionProps> = ({ useLevaStore = false, value = demoError }) => {
    let control: ChatItemProps['error'] | any;
    let store;

    if (useLevaStore) {
        store = useCreateStore();
        control = useControls(
            {
                description: 'Finished inferring TextToImage. Copy the JSON file below and use it in Stable Diffusion',
                message: 'Done!',
                type: {
                    options: ['success', 'info', 'warning', 'error'],
                    value: 'success',
                },
            },
            { store },
        );
    } else {
        control = {
            description: 'Finished inferring TextToImage. Copy the JSON file below and use it in Stable Diffusion',
            message: 'Done!',
            type: 'success',
        };
    }

    return (
            <ChatItem
                avatar={avatar}
                error={control}
                errorMessage={
                    <Highlighter copyButtonSize={'small'} language={'json'} type={'pure'}>
                        {JSON.stringify(value ? value : demoError, null, 2)}
                    </Highlighter>
                }
            />
    );
};

export default Chat;
