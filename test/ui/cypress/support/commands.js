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
  const password = Cypress.env('OCP_PASSWORD')
  expect(password).to.match(/^(?:[a-zA-Z0-9]{5}-){3}[a-zA-Z0-9]{5}$/)

  cy.visit('/')
  cy.url().should('include', '/oauth/authorize')
  cy.contains('Log in with')
  cy.contains('kube:admin').click()
  cy.url().should('include', '/login/kube:admin')

  cy.get('#inputUsername')
    .type('kubeadmin')
    .should('have.value', 'kubeadmin')

  cy.get('#inputPassword')
    .type(password)
    .should('have.value', password)
  cy.get('button[type=submit]').click()

  cy.contains('kube:admin')
})
