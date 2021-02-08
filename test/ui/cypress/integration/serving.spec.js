describe('OCP UI for Serverless', () => {

  class ShowcaseKservice {
    makeRequest() {
      cy.get('a.co-external-link')
      .scrollIntoView()
      .should('have.attr', 'href')
      .and('include', 'showcase')
      .then((href) => {
        cy.request({ method: 'OPTIONS', url: href, retryOnStatusCodeFailure: true }).then((response) => {
          expect(response.body).to.have.property('artifact-id', 'knative-serving-showcase')
        })
      })  
    }

    checkScale(scale) {
      cy.get('div.pf-topology-container__with-sidebar div.odc-revision-deployment-list__pod svg tspan')
        .invoke('text')
        .should((text) => {
          expect(text).to.eq(`${scale}`)
        })
    }
  }

  const showcaseKsvc = new ShowcaseKservice()
  

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
      cy.get('div.pf-topology-content').contains('showcase').click()
      cy.contains('Location:')
      cy.contains('Running')
      showcaseKsvc.makeRequest()
      showcaseKsvc.checkScale(1)
      cy.wait(60_000) // 60sec.
      showcaseKsvc.checkScale(0)
      showcaseKsvc.makeRequest()
      showcaseKsvc.checkScale(1)
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
