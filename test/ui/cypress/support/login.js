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

  cy.visit(`/topology/ns/${namespace}?view=graph`)
  cy.contains('Topology')
  cy.contains(username)
  cy.get('body').then(($body) => {
    if ($body.find('[data-test="guided-tour-modal"]').length) {
      cy.contains('Skip tour').click()
    }
  })

})
