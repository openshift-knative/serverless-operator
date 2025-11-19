const {defineConfig} = require('cypress')
const path = require('path')

// Determine base directory for test results
const artifactsDir = path.join(process.env.ARTIFACTS || 'results', 'ui')

module.exports = defineConfig({
  defaultCommandTimeout: 60_000,
  reporter: 'cypress-multi-reporters',
  reporterOptions: {
    reporterEnabled: 'spec, mocha-junit-reporter',
    mochaJunitReporterReporterOptions: {
      mochaFile: path.join(artifactsDir, 'junit-[hash].xml'),
    },
  },
  retries: {
    runMode: 2,
    openMode: 0,
  },
  // Configure artifact directories
  screenshotsFolder: path.join(artifactsDir, 'screenshots'),
  videosFolder: path.join(artifactsDir, 'videos'),
  e2e: {
    // Disable web security to allow cross-origin OAuth flow (console → oauth-openshift → console)
    chromeWebSecurity: false,
    video: true,
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
