const config = {
    presets: [
        ['@babel/preset-env', {
            targets: {
                chrome: 100,
                firefox: 100,
                edge: 100,
                safari: 15,
            },
            modules: false,
            corejs: 3,
            debug: false,
            useBuiltIns: 'usage',
            shippedProposals: true,
        }],
        ['@babel/preset-react', {
            runtime: 'automatic',
        }],
        ['@babel/preset-typescript', {
            allExtensions: true,
            isTSX: true,
        }],
        ['@emotion/babel-preset-css-prop'],
    ],
    plugins: [
        'babel-plugin-typescript-to-proptypes',
    ],
};

// Jest needs module transformation.
config.env = {
    test: {
        presets: config.presets,
        plugins: config.plugins,
    },
};
config.env.test.presets[0][1].modules = 'auto';

module.exports = config;
