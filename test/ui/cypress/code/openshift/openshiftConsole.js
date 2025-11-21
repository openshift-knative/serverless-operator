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
        const selector = selectors.drawer + selectors.closeButtonSelector
        if (selectors.checkIsOpen($drawer)) {
          cy.log('Closing sidebar')
          cy.get(selector).click()
        }
      })
  }

  /**
   * Returns CSS selectors for topology sidebar based on OCP version.
   *
   * Version Timeline:
   * - OCP 4.14:      PatternFly v4, old drawer, data-test-id
   * - OCP 4.15-4.18: PatternFly v5, new drawer, data-test-id
   * - OCP 4.19.x:    PatternFly v6, new drawer, data-test-id (migration in progress)
   * - OCP 4.20+:     PatternFly v6, new drawer, data-test (migration complete)
   *
   * @returns {{checkIsOpen: function(JQuery<HTMLElement>): boolean, drawer: string, closeButtonSelector: string}}
   *   Object containing sidebar selectors:
   *   - checkIsOpen: Function to check if drawer is open
   *   - drawer: CSS selector for the drawer panel element
   *   - closeButtonSelector: CSS selector for the close button (appended to drawer selector)
   */
  sidebarSelectors() {
    const version = environment.ocpVersion()
    
    // OCP 4.14 and earlier: PatternFly v4
    if (version.satisfies('<=4.14')) {
      return {
        checkIsOpen: (drawer) => drawer.hasClass('pf-m-expanded'),
        drawer: '.odc-topology .pf-c-drawer',
        closeButtonSelector: ' button[data-test-id=sidebar-close-button]',
      }
    }
    
    // OCP 4.15-4.18: PatternFly v5
    if (version.satisfies('<=4.18')) {
      return {
        checkIsOpen: (drawer) => drawer.find('.pf-topology-resizable-side-bar').length > 0,
        drawer: '.pf-v5-c-drawer__panel.ocs-sidebar-index',
        closeButtonSelector: ' button[data-test-id=sidebar-close-button]',
      }
    }
    
    // OCP 4.19.x: PatternFly v6 with old data-test-id attribute
    if (version.satisfies('<4.20')) {
      return {
        checkIsOpen: (drawer) => drawer.find('.pf-topology-resizable-side-bar').length > 0,
        drawer: '.pf-v6-c-drawer__panel.ocs-sidebar-index',
        closeButtonSelector: ' button[data-test-id=sidebar-close-button]',
      }
    }
    
    // OCP 4.20+: PatternFly v6 with new data-test attribute
    return {
      checkIsOpen: (drawer) => drawer.find('.pf-topology-resizable-side-bar').length > 0,
      drawer: '.pf-v6-c-drawer__panel.ocs-sidebar-index',
      closeButtonSelector: ' button[data-test=sidebar-close-button]',
    }
  }
}

export default OpenshiftConsole
