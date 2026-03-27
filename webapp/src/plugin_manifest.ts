import rawManifest from '../../plugin.json';

const manifest = {
    ...rawManifest,
    version: (rawManifest as {version?: string}).version || '0.0.0-dev',
};

export default manifest;
