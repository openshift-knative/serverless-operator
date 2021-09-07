import SemverResolver from "./semverResolver";

class Environment {
  ocpVersion() {
    if (!Environment.__ocpVersion) {
      Environment.__ocpVersion = new SemverResolver(Cypress.env('OCP_VERSION'))
    }
    return Environment.__ocpVersion
  }

  loginProvider() {
    return Cypress.env('OCP_LOGIN_PROVIDER')
  }

  username() {
    return Cypress.env('OCP_USERNAME')
  }

  password() {
    return Cypress.env('OCP_PASSWORD')
  }

  namespace() {
    return Cypress.env('TEST_NAMESPACE')
  }
}

export default Environment
