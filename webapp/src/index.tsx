// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import manifest from 'plugin_manifest';
import type {Store} from 'redux';

import type {GlobalState} from '@mattermost/types/store';

import AdminConsolePanel from 'components/admin_console_panel';
import type {PluginRegistry} from 'types/mattermost-webapp';

const operationsSettingKey = 'OperationsPanelPlaceholder';

export default class Plugin {
    public async initialize(registry: PluginRegistry, _store: Store<GlobalState>) {
        registry.registerAdminConsoleCustomSetting(operationsSettingKey, AdminConsolePanel, {showTitle: false});
    }
}

declare global {
    interface Window {
        registerPlugin(pluginId: string, plugin: Plugin): void;
    }
}

window.registerPlugin(manifest.id, new Plugin());
