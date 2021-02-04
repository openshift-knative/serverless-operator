describe('OCP UI for Serverless', () => {

  it('can deploy kservice', () => {
    describe('with authenticated via Web Console', () => {
      cy.login()
    })
    describe('deploy kservice from image', () => {
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
        .type('demoapp')
      cy.get('input#form-input-name-field')
        .clear()
        .type('showcase')
      cy.get('button[type=submit]').click()
    })
    describe('check availibility of kservice', () => {
      cy.contains('No Revisions')
      cy.contains('demoapp')
      cy.visit('/topology/ns/default/list')
      cy.contains('showcase').click()
      cy.contains('Location:')
      cy.contains('Running')
      cy.get('a.co-external-link')
        .scrollIntoView()
        .should('have.attr', 'href')
        .and('include', 'showcase')
        .then((href) => {
          cy.request({ method: 'OPTIONS', url: href, retryOnStatusCodeFailure: true }).then((response) => {
            expect(response.body).to.have.property('artifact-id', 'knative-serving-showcase')
          })
        })
    })
    describe('remove kservice', () => {

      cy.visit('/topology/ns/default/list')
      cy.contains('demoapp').click()
      cy.contains('Actions').click()
      cy.contains('Delete Application').click()
      cy.get('input#form-input-resourceName-field')
        .type('demoapp{enter}')
      cy.contains('No resources found')
    })
  })
})
