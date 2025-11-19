const {defineConfig} = require('cypress')
const path = require('path')
const fs = require('fs')

// Determine base directory for test results
const artifactsDir = path.join(process.env.ARTIFACTS || 'results', 'ui')

/**
 * Safely delete a file, ignoring errors if file doesn't exist
 * @param {string} filePath - Path to file to delete
 */
function safeUnlink(filePath) {
  try {
    fs.unlinkSync(filePath)
  } catch (err) {
    // Ignore errors (file may not exist)
  }
}

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
      // Delete videos for specs without failing or retried tests
      // This saves artifact storage space by only keeping videos of failed tests
      on('after:spec', (spec, results) => {
        if (results && results.video) {
          // Check if any test attempt failed
          const hasFailures = results.tests.some((test) =>
            test.attempts.some((attempt) => attempt.state === 'failed')
          )
          // Delete video if all tests passed
          if (!hasFailures) {
            safeUnlink(results.video)
            // Also delete compressed video if it exists
            safeUnlink(results.video.replace('.mp4', '-compressed.mp4'))
          }
        }
      })

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
