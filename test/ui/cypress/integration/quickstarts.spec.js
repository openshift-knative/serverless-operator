describe('OCP UI for Serverless', () => {
  it('have Serverless quickstarts', () => {
    cy.login()
    cy.visit('/settings/cluster')

    cy.get('span[data-test-id=cluster-version]').invoke('text').then((version) => {
      cy.log(`OCP version: ${version}`)
      cy.visit('/quickstart')
      cy.semver().then((semver) => {
        if (semver.satisfies(version, '>=4.7.0')) {
          cy.log('OCP version is >=4.7.0')
          cy.contains('Setting up Serverless')
        } else {
          cy.log('OCP version is <4.7.0')
        }
        cy.contains('Exploring Serverless applications')
      })
    })
  })
})
