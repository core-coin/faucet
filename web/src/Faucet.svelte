<script>
  import { onMount } from 'svelte';
  import { getAddress } from './address.js';
  import { setDefaults as setToast, toast } from 'bulma-toast';

  let address = null;
  let faucetInfo = {
    account: 'AB000000000000000000000000000000000000000000',
    network: 'devin',
    payout: 1,
  };

  $: document.title = `XAB ${capitalize(faucetInfo.network)} Faucet`;

  onMount(async () => {
    const res = await fetch('/api/info');
    faucetInfo = await res.json();
  });

  setToast({
    position: 'bottom-center',
    dismissible: true,
    pauseOnHover: true,
    closeOnClick: false,
    animate: { in: 'fadeIn', out: 'fadeOut' },
  });

  async function handleRequest() {
    try {
      address = getAddress(address);
    } catch (error) {
      toast({ message: error.reason, type: 'is-warning' });
      return;
    }

    let formData = new FormData();
    formData.append('address', address);
    const res = await fetch('/api/claim', {
      method: 'POST',
      body: formData,
    });
    let message = await res.text();
    let type = res.ok ? 'is-success' : 'is-warning';
    toast({ message, type });
  }

  function capitalize(str) {
    const lower = str.toLowerCase();
    return str.charAt(0).toUpperCase() + lower.slice(1);
  }
</script>

<main>
  <section class="hero bcg is-fullheight">
    <nav class="navbar">
      <div class="navbar-menu">
        <div class="navbar-start">
          <span class="navbar-item">
            <a class="button is-primary is-outlined" href="https://coreblockchain.cc" target="_blank">Blockchain</a>
          </span>
        </div>
        <div class="navbar-end">
          <span class="navbar-item">
            <a class="button is-info is-outlined" href="https://github.com/core-coin/faucet" target="_blank">Source code</a>
          </span>
        </div>
      </div>
    </nav>

    <div class="hero-body">
      <div class="container has-text-centered">
        <div class="column">
          <h1 class="title">
            Receive {faucetInfo.payout} XAB per request
          </h1>
          <h2 class="subtitle">
            Serving from {faucetInfo.account}
          </h2>
          <div class="box">
            <div class="field is-grouped">
              <p class="control is-expanded">
                <input
                  bind:value={address}
                  class="input is-medium"
                  type="text"
                  placeholder="Enter your wallet"
                />
              </p>
              <p class="control">
                <button
                  on:click={handleRequest}
                  class="button is-primary is-medium"
                >
                  Request
                </button>
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</main>

<style>
  .title, h1, h2 {
    color: #d86f42;
  }
  .hero.bcg {
    background: url('/background.png') no-repeat center center fixed;
    -webkit-background-size: cover;
    -moz-background-size: cover;
    -o-background-size: cover;
    background-size: cover;
  }
  .hero .subtitle {
    padding: 3rem 0;
    line-height: 1.5;
  }
  .box {
    border-radius: 1em;
    background-color: rgba(255,255,255,0.7);
  }
  input::placeholder {
    color: #363636;
    opacity: 1;
  }

  input:-ms-input-placeholder {
    color: #363636;
  }

  input::-ms-input-placeholder {
    color: #363636;
  }
</style>
