describe('OCP UI for Serverless', () => {

  it('can deploy kservice', () => {
    context('with authenticated via Web Console', () => {
      cy.login()
    })
    context('deploy Knative Service from image', () => {
      cy.visit('/add/ns/default')
      cy.contains('Knative Channel')
      cy.contains('Event Source')
      cy.visit('/deploy-image/ns/default')
      cy.get('input[name=searchTerm]')
        .type('quay.io/cardil/knative-serving-showcase:2-send-event')
      cy.contains('Validated')
      cy.get('input#form-radiobutton-resources-knative-field').check()
      cy.get('input#form-checkbox-route-create-field').check()
      cy.get('input#form-input-application-name-field')
        .clear()
        .type('showcase-app')
      cy.get('input#form-input-name-field')
        .clear()
        .type('showcase')
      try {
        cy.get('button[type=submit]').click()
        cy.contains('No Revisions')
        cy.contains('showcase-app')
      } finally {
        cy.get('a.odc-topology__view-switcher').click()
        cy.contains('showcase-app').click()
        cy.contains('Actions').click()
        cy.contains('Delete Application').click()
        cy.get('input#form-input-resourceName-field')
          .type('showcase-app{enter}')
        cy.contains('No resources found')
      }
    })
  })
})
