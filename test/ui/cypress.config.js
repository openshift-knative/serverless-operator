const {defineConfig} = require('cypress')

module.exports = defineConfig({
  defaultCommandTimeout: 60_000,
  reporter: 'cypress-multi-reporters',
  reporterOptions: {
    configFile: 'reporter-config.json',
  },
  retries: {
    runMode: 2,
    openMode: 0,
  },
  e2e: {
    setupNodeEvents(on, config) {
      config.env.TEST_NAMESPACE = process.env.TEST_NAMESPACE || 'default'
      config.env.OCP_LOGIN_PROVIDER = process.env.OCP_LOGIN_PROVIDER || 'kube:admin'
      config.env.OCP_VERSION = process.env.OCP_VERSION || '0.0.0'
      config.env.OCP_USERNAME = process.env.OCP_USERNAME || 'kube:admin'
      config.env.OCP_PASSWORD = process.env.OCP_PASSWORD
      if (!config.env.OCP_PASSWORD) {
        throw new Error('OCP_PASSWORD is not set')
      }

      return config
    },
  },
})
