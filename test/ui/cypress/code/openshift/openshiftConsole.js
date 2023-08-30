import Environment from '../environment'
import Kubernetes from '../kubernetes'

const environment = new Environment()
const k8s = new Kubernetes()

class OpenshiftConsole {
  login() {
    cy.log('Login to OCP')
    const loginProvider = environment.loginProvider()
    const username = environment.username()
    const password = environment.password()
    const namespace = environment.namespace()

    expect(password).to.match(/^.{3,}$/)

    cy.on('uncaught:exception', (err) => {
      // returning false here prevents Cypress from failing the test
      return !(err.hasOwnProperty('response') && err.response.status === 401)
    })

    cy.visit('/')
    cy.url().should('include', 'oauth-openshift')
    cy.url().then((url) => {
      if (loginProvider !== '' && new URL(url).pathname !== '/login') {
        cy.url().should('include', '/oauth/authorize')
        cy.contains('Log in with')
        cy.contains(loginProvider).click()
        cy.url().should('include', `/login/${loginProvider}`)
      }
    })

    cy.get('#inputUsername')
      .type(username)
      .should('have.value', username)

    cy.get('#inputPassword')
      .type(password)
      .should('have.value', password)
    cy.get('button[type=submit]').click()
    cy.url().should('not.include', 'oauth-openshift')

    cy.on('uncaught:exception', () => {
      // restore exception processing
      return true
    })

    k8s.ensureNamespaceExists(namespace)

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

  closeSidebar() {
    const selectors = this.sidebarSelectors()
    cy.get(selectors.drawer)
      .then(($drawer) => {
        if ($drawer.hasClass(selectors.expandedCls)) {
          cy.log('Closing sidebar')
          cy.get(selectors.closeBtn).click()
        }
      })
  }

  sidebarSelectors() {
    let dataTestAction = 'Delete application'
    if (environment.ocpVersion().satisfies('<4.11')) {
      dataTestAction = 'Delete Application'
    }
    if (environment.ocpVersion().satisfies('<4.10')) {
      return {
        drawer: '.odc-topology .pf-topology-container',
        expandedCls: 'pf-topology-container__with-sidebar--open',
        closeBtn: '.odc-topology .pf-topology-container .pf-topology-side-bar button.close',
        deleteApplicationBtn: `button[data-test-action="${dataTestAction}"]`
      }
    }
    return {
      drawer: '.odc-topology .pf-c-drawer',
      expandedCls: 'pf-m-expanded',
      closeBtn: '.odc-topology .pf-c-drawer button[data-test-id=sidebar-close-button]',
      deleteApplicationBtn: `li[data-test-action="${dataTestAction}"] button`
    }
  }
}

export default OpenshiftConsole
