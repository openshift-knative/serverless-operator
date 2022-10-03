import SemverResolver from './semverResolver'

class Environment {
  ocpVersion() {
    return new SemverResolver(Cypress.env('OCP_VERSION'))
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
