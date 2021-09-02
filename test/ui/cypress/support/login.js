Cypress.Commands.add('login', () => {
  const loginProvider = Cypress.env('OCP_LOGIN_PROVIDER')
  const username = Cypress.env('OCP_USERNAME')
  const password = Cypress.env('OCP_PASSWORD')
  const namespace = Cypress.env('TEST_NAMESPACE')
  expect(password).to.match(/^.{3,}$/)

  cy.on('uncaught:exception', () => {
    return false
  })
  cy.visit('/')
  cy.url().should('include', '/oauth/authorize')
  cy.contains('Log in with')
  cy.contains(loginProvider).click()
  cy.url().should('include', `/login/${loginProvider}`)

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
    cy.ocpVersion().then(version => {
      let selector = '[data-test="guided-tour-modal"]'
      if (version.satisfies('>=4.9')) {
        selector = '#guided-tour-modal'
      }
      cy.log(`Guided Tour modal selector used: ${selector}`)
      const modal = $body.find(selector)
      if (modal.length) {
        cy.contains('Skip tour').click()
      }
    })
  })

})
