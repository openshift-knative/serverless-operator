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
      let selector = '#guided-tour-modal'
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
        const selector = selectors.drawer +
          ' button[data-test-id=sidebar-close-button]'
        if (selectors.checkIsOpen($drawer)) {
          cy.log('Closing sidebar')
          cy.get(selector).click()
        }
      })
  }

  sidebarSelectors() {
    if (environment.ocpVersion().satisfies('<=4.14')) {
      return {
        /**
         * @param drawer {JQuery<HTMLElement>}
         * @returns {boolean}
         */
        checkIsOpen: function (drawer) {
          return drawer.hasClass('pf-m-expanded')
        },
        drawer: '.odc-topology .pf-c-drawer',
      }
    }
    return {
      /**
       * @param drawer {JQuery<HTMLElement>}
       * @returns {boolean}
       */
      checkIsOpen: function (drawer) {
        return drawer.find('.pf-topology-resizable-side-bar').length > 0
      },
      drawer: '.pf-v5-c-drawer__panel.ocs-sidebar-index',
    }
  }
}

export default OpenshiftConsole
