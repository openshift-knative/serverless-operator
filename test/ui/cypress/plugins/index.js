/// <reference types="cypress" />
// ***********************************************************
// This example plugins/index.js can be used to load plugins
//
// You can change the location of this file or turn off loading
// the plugins file with the 'pluginsFile' configuration option.
//
// You can read more here:
// https://on.cypress.io/plugins-guide
// ***********************************************************

// This function is called when a project is opened or re-opened (e.g. due to
// the project's config changing)

/**
 * @type {Cypress.PluginConfig}
 */
module.exports = (on, config) => {
  // `on` is used to hook into various events Cypress emits
  // `config` is the resolved Cypress config
  config.env.TEST_NAMESPACE = process.env.TEST_NAMESPACE || 'default'
  config.env.OCP_LOGIN_PROVIDER = process.env.OCP_LOGIN_PROVIDER || 'kube:admin'
  config.env.OCP_VERSION = process.env.OCP_VERSION || '0.0.0'
  config.env.OCP_USERNAME = process.env.OCP_USERNAME || 'kube:admin'
  config.env.OCP_PASSWORD = process.env.OCP_PASSWORD

  require('cypress-terminal-report/src/installLogsPrinter')(on)

  return config
}
