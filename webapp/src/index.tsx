import type {Post} from '@mattermost/types/posts';
import type {GlobalState} from '@mattermost/types/store';

import {createVoicePostAttachmentComponent} from './components/voice_post_attachment';
import {VoiceRecorderAction} from './components/voice_recorder_action';
import manifest from './manifest';
import type {PluginRegistry} from './types/mattermost-webapp';

import './styles/voice.scss';

type MinimalStore = {
    getState(): GlobalState;
};

function getPostById(store: MinimalStore, postId: string): Post | undefined {
    return store.getState().entities?.posts?.posts?.[postId];
}

export default class VoicePlugin {
    public initialize(registry: PluginRegistry, store: MinimalStore) {
        registry.registerPostEditorActionComponent(VoiceRecorderAction);
        registry.registerPostMessageAttachmentComponent(
            createVoicePostAttachmentComponent((postId) => getPostById(store, postId)),
        );
    }
}

declare global {
    interface Window {
        registerPlugin(pluginId: string, plugin: VoicePlugin): void;
    }
}

window.registerPlugin(manifest.id, new VoicePlugin());
