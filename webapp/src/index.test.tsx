export {};

describe('webapp plugin registration', () => {
    test('registers the admin console custom panel hook', async () => {
        const registerPlugin = jest.fn();
        (globalThis as any).window = {registerPlugin};

        let PluginClass: any;
        let manifest: any;

        jest.isolateModules(() => {
            manifest = require('plugin_manifest').default;
            PluginClass = require('./index').default;
        });

        expect(registerPlugin).toHaveBeenCalledWith(manifest.id, expect.any(PluginClass));

        const registerAdminConsoleCustomSetting = jest.fn();
        const plugin = new PluginClass();
        await plugin.initialize({
            registerAdminConsoleCustomSetting,
        }, {});

        expect(registerAdminConsoleCustomSetting).toHaveBeenCalledWith(
            'OperationsPanelPlaceholder',
            expect.any(Function),
            {showTitle: false},
        );
    });
});
