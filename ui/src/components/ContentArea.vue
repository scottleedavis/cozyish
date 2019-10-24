<template>
  <div>
  <div v-if="samples.length > 0">
  <div class="gallery">
    <div  v-bind:key="sample" v-for="sample in samples">
      <div class="item" v-if="sample.nsfw_score.toPrecision(3) != 0.00">
       <img width="200px" class="image" :src="sample.image">
       <div class="info">
          <p>{{ sample.tags.join(", ") }}</p>
          <label for="nsfw">NSFW score</label>
          <p id="nsfw">{{ sample.nsfw_score.toPrecision(3) }}</p>
          <label for="exif">EXIF</label>
          <textarea id="exif" v-model="sample.exif"></textarea>
          <label for="steg">steganography</label>
          <textarea id="steg" v-model="sample.steganography"></textarea>
       </div>
      </div>
    </div>
  </div>
  </div>
  <div class="empty" v-if="samples.length === 0">
    <div>
      No results found.
    </div>
  </div>
  </div>
</template>
 
<script>
  import { EventBus } from "../event-bus.js";

  var samples = [];
  console.log(samples)

  export default {
    name: 'ContentArea',
    created: function() {
      EventBus.$on('samples_ready', this.samplesReady)
    },
    data: function () {
      return {
        samples: [],
        index: null
      };
    },

    methods: {
      samplesReady : function(data){
        data = data.map(d => {
          if (d.exif.length > 0) {
            d.exif = d.exif.map(e => {
                var key = Object.keys(e)[0]
                var value = Object.values(e)[0]
                return key+"="+value;
            })
          }
          return d;
        });
        this.samples = data;
      }
    },
  }
</script> 
 
<style scoped>
  .gallery {
    display: flex;
    flex-wrap: wrap;
  }
  .item {
    margin: 50px;
    width: 300px;
    display: flex;
    flex-direction: column;
  }
  .image {
    float: left;
    background-size: cover;
    background-repeat: no-repeat;
    background-position: center center;
    border: 1px solid #ebebeb;
    margin-left: 50px;
  }
  .info {
    flex-direction: column;
    display: flex;
  }
  label {
        font-weight: bold;
  }
  .empty {
    margin: 100px;
    font-size: -webkit-xxx-large;
  }
</style> 
 