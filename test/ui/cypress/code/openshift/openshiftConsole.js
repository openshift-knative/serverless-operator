import Environment from "../environment";

const environment = new Environment()

class OpenshiftConsole {
  login() {
    const loginProvider = environment.loginProvider()
    const username = environment.username()
    const password = environment.password()
    const namespace = environment.namespace()

    expect(password).to.match(/^.{3,}$/)

    cy.on('uncaught:exception', () => {
      return false
    })
    cy.visit('/')
    if (loginProvider !== '') {
      cy.url().should('include', '/oauth/authorize')
      cy.contains('Log in with')
      cy.contains(loginProvider).click()
      cy.url().should('include', `/login/${loginProvider}`)
    }

    cy.get('#inputUsername')
      .type(username)
      .should('have.value', username)

    cy.get('#inputPassword')
      .type(password)
      .should('have.value', password)
    cy.get('button[type=submit]').click()

    cy.visit(`/add/ns/${namespace}?view=graph`)
    cy.get('#content').contains('Add')
    cy.get('body').then(($body) => {
      let selector = '[data-test="guided-tour-modal"]'
      if (environment.ocpVersion().satisfies('>=4.9')) {
        selector = '#guided-tour-modal'
      }
      cy.log(`Guided Tour modal selector used: ${selector}`)
      const modal = $body.find(selector)
      if (modal.length) {
        cy.contains('Skip tour').click()
      }
    })
  }
}

export default OpenshiftConsole
