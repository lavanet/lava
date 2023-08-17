const useESModules = !!process.env.MODULE;

module.exports = (api) => {
  api.cache(() => process.env.MODULE);
  return {
    plugins: [
      ['@babel/transform-runtime', { useESModules }],
      '@babel/proposal-object-rest-spread',
      '@babel/proposal-class-properties',
      '@babel/plugin-proposal-numeric-separator',
      '@babel/plugin-proposal-nullish-coalescing-operator',
      '@babel/plugin-proposal-optional-chaining',
      '@babel/proposal-export-default-from'
    ],
    presets: useESModules ? ['@babel/typescript'] : ['@babel/typescript', '@babel/env']
  };
};