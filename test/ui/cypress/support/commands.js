// ***********************************************
// This example commands.js shows you how to
// create various custom commands and overwrite
// existing commands.
//
// For more comprehensive examples of custom
// commands please read more here:
// https://on.cypress.io/custom-commands
// ***********************************************
//
//
// -- This is a parent command --
// Cypress.Commands.add("login", (email, password) => { ... })
//
//
// -- This is a child command --
// Cypress.Commands.add("drag", { prevSubject: 'element'}, (subject, options) => { ... })
//
//
// -- This is a dual command --
// Cypress.Commands.add("dismiss", { prevSubject: 'optional'}, (subject, options) => { ... })
//
//
// -- This will overwrite an existing command --
// Cypress.Commands.overwrite("visit", (originalFn, url, options) => { ... })

Cypress.Commands.add('login', () => {
  const loginProvider = Cypress.env('OCP_LOGIN_PROVIDER')
  const username = Cypress.env('OCP_USERNAME')
  const password = Cypress.env('OCP_PASSWORD')
  expect(password).to.match(/^.{3,}$/)

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

  cy.visit('/dashboards')
  cy.contains(username)
})
