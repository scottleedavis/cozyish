<template>
  <div id="app">
  <b-navbar toggleable="lg" type="dark" variant="info">
    <b-navbar-brand href="#">Cozyish</b-navbar-brand>
    <b-navbar-toggle target="nav-collapse"></b-navbar-toggle>
    <b-collapse id="nav-collapse" is-nav>
      <b-navbar-nav>
        <b-nav-item href="#">Images</b-nav-item>
      </b-navbar-nav>
      <!-- <b-nav-item-dropdown text="Video" right>
        <b-dropdown-item href="#"><iframe src="http://localhost:3000"></iframe></b-dropdown-item>
      </b-nav-item-dropdown> -->
      <b-nav-form>
        <b-form-input size="sm" @keydown.native="siteUrlKeydown" v-model="url" class="mr-sm-2" placeholder="Site URL"></b-form-input>
        <b-button size="sm" class="my-2 my-sm-0" @click=crawl>Crawl</b-button>
      </b-nav-form>
      <b-navbar-nav class="ml-auto">
        <b-nav-form>
          <b-button size="sm" class="my-2 my-sm-0" @click=refresh>Refresh</b-button>
          <b-navbar-nav>
            <b-nav-text>-</b-nav-text>
          </b-navbar-nav>
          <b-button size="sm" class="my-2 my-sm-0" @click=deleteAll>Delete All</b-button>
        </b-nav-form>
      </b-navbar-nav>
    </b-collapse>
  </b-navbar>
  <ContentArea />
  </div>
</template>

<script>
import ContentArea from './components/ContentArea.vue'
import axios from 'axios';
import { EventBus } from "./event-bus.js";

export default {
  name: 'app',
  components: {
    ContentArea
  },
  mounted() {
    this.refreshAuto()
  },
  data() {
    return {
      url: '',
      samples: [],
    }
  },
  methods: {
    refresh: function() {
      this.getAll()
    },
    refreshAuto: function() {
      var that = this;
      setInterval(function() {
        that.getAll();
      }, 5000);
    },
    deleteAll : function(){
      axios.get('/api/image/delete').then(() =>  this.getAll() )
    },
    getAll: function() {
      EventBus.$emit("samples_ready", []); 
      if (this.url === "") 
        return;
      axios.get('/api/image?url='+this.url).then(response => {
          if (response.data.length != this.samples) {
            EventBus.$emit("samples_ready", response.data); 
            this.samples = response.data;
          }
      })
    },
    crawl: function() {
      axios.get('/api?url='+this.url).then(() => this.getAll())
    },
    siteUrlKeydown: function(event) {
      if (event.which === 13) {
        axios.get('/api?url='+this.url).then(() => this.getAll())
      }
    }
  },

}
</script>

<style>
#app {
  font-family: 'Avenir', Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  color: #2c3e50;
}
</style>
