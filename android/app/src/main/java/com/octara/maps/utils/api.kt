package com.octara.maps.utils

import com.google.gson.Gson
import com.google.gson.GsonBuilder
import com.google.gson.TypeAdapter
import com.google.gson.stream.JsonReader
import com.google.gson.stream.JsonToken
import com.google.gson.stream.JsonWriter
import com.octara.maps.types.Accessibility
import com.octara.maps.types.Departure
import com.octara.maps.types.FavoriteDepartureInfo
import com.octara.maps.types.JourneyQuery
import com.octara.maps.types.JourneyResponse
import com.octara.maps.types.LegType
import com.octara.maps.types.ParentStop
import com.octara.maps.types.Position
import com.octara.maps.types.Route
import com.octara.maps.types.RouteType
import com.octara.maps.types.Stop
import com.octara.maps.types.StopType
import com.octara.maps.types.TimeType
import com.octara.maps.types.TrainPosition
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.net.HttpURLConnection
import java.net.URL
import java.net.URLEncoder

class Api {
    companion object {
        const val BASE_URL = "https://omaps-calc.orionhost.app/api"
    }

    val gson: Gson = GsonBuilder()
        .registerTypeAdapter(StopType::class.java, object : TypeAdapter<StopType>() {
            override fun write(out: JsonWriter, value: StopType?) {
                out.value(value?.value ?: 0)
            }

            override fun read(reader: JsonReader): StopType {
                if (reader.peek() == JsonToken.NULL) {
                    reader.nextNull()
                    return StopType.UNKNOWN
                }
                val intVal = reader.nextInt()
                return StopType.fromValue(intVal)
            }
        })
        .registerTypeAdapter(RouteType::class.java, object : TypeAdapter<RouteType>() {
            override fun write(out: JsonWriter, value: RouteType?) {
                out.value(value?.value ?: 3)
            }

            override fun read(reader: JsonReader): RouteType {
                if (reader.peek() == JsonToken.NULL) {
                    reader.nextNull()
                    return RouteType.BUS
                }
                val intVal = reader.nextInt()
                return RouteType.fromValue(intVal)
            }
        })
        .registerTypeAdapter(LegType::class.java, object : TypeAdapter<LegType>() {
            override fun write(out: JsonWriter, value: LegType?) {
                out.value(value?.value ?: 0)
            }

            override fun read(reader: JsonReader): LegType {
                if (reader.peek() == JsonToken.NULL) {
                    reader.nextNull()
                    return LegType.WALK
                }
                val intVal = reader.nextInt()
                return LegType.fromValue(intVal)
            }
        })
        .registerTypeAdapter(Accessibility::class.java, object : TypeAdapter<Accessibility>() {
            override fun write(out: JsonWriter, value: Accessibility?) {
                out.value(value?.value ?: 0)
            }

            override fun read(reader: JsonReader): Accessibility {
                if (reader.peek() == JsonToken.NULL) {
                    reader.nextNull()
                    return Accessibility.UNKNOWN
                }
                val intVal = reader.nextInt()
                return Accessibility.fromValue(intVal)
            }
        })
        .registerTypeAdapter(TrainPosition::class.java, object : TypeAdapter<TrainPosition>() {
            override fun write(out: JsonWriter, value: TrainPosition?) {
                out.value(value?.value ?: 0)
            }

            override fun read(reader: JsonReader): TrainPosition {
                if (reader.peek() == JsonToken.NULL) {
                    reader.nextNull()
                    return TrainPosition.UNKNOWN
                }
                val intVal = reader.nextInt()
                return TrainPosition.fromValue(intVal)
            }
        })
        .create()

    inline fun <reified T> fetchGET(url: String): T {
        android.util.Log.d("OctaraApi", "Fetching GET: $url")
        val connection = (URL(url).openConnection() as HttpURLConnection).apply {
            requestMethod = "GET"
            connectTimeout = 10_000
            readTimeout = 10_000
        }

        return try {
            val responseCode = connection.responseCode
            android.util.Log.d("OctaraApi", "Response code $responseCode for $url")
            if (responseCode !in 200..299) {
                val errText = try { connection.errorStream?.bufferedReader()?.use { it.readText() } } catch (_: Exception) { null }
                android.util.Log.e("OctaraApi", "GET failed ($responseCode): $errText")
                throw IllegalStateException("GET $url failed ($responseCode): $errText")
            }

            val type = object : com.google.gson.reflect.TypeToken<T>() {}.type
            val result = connection.inputStream.bufferedReader().use { reader ->
                gson.fromJson<T>(reader, type)
            } ?: throw IllegalStateException("json is null")
            android.util.Log.d("OctaraApi", "Parsed successfully: ${result?.javaClass?.simpleName}")
            result
        } catch (e: Exception) {
            android.util.Log.e("OctaraApi", "Exception during GET $url", e)
            throw e
        } finally {
            connection.disconnect()
        }
    }

    private data class StopResponse(
        val total: Int,
        val stops: List<ParentStop>?
    )

    suspend fun searchStopsByName(
        name: String,
        limit: Int = 10,
        cpos: Pair<Float, Float>
    ): List<ParentStop> = withContext(Dispatchers.IO) {
        val encodedQuery = URLEncoder.encode(name, "UTF-8")
        val url = "$BASE_URL/stops/autocomplete?q=$encodedQuery&maxstops=$limit&lat=${cpos.first}&lon=${cpos.second}"

        val response: StopResponse = fetchGET(url)
        response.stops ?: emptyList()
    }

    public data class StopInfoResponse(
        val stop: Stop,
        val departures: List<Departure>?,
        val routes: List<Route>?
    )

    suspend fun getStopById(
        id: String,
    ): StopInfoResponse = withContext(Dispatchers.IO) {
        val url = "$BASE_URL/stops/$id"

        val response: StopInfoResponse = fetchGET(url)
        response
    }

    suspend fun getNextDeparturesOfMultipleStops(list: List<Pair<String, String>>): Map<Pair<String, String>, Map<String, List<Int>>> =
        withContext(Dispatchers.IO) {
            val details = getNextDepartureDetailsOfMultipleStops(list)
            details.mapValues { (_, byHeadsign) ->
                byHeadsign.mapValues { (_, items) -> items.map { it.wait_sec } }
            }
        }

    suspend fun getNextDepartureDetailsOfMultipleStops(list: List<Pair<String, String>>): Map<Pair<String, String>, Map<String, List<FavoriteDepartureInfo>>> =
        withContext(Dispatchers.IO) {
            if (list.isEmpty()) return@withContext emptyMap()
            try {
                val url = "$BASE_URL/nextdepartures?details=true&stops=${list.joinToString(",") { "${it.first}:${it.second}" }}"
                val response: Map<String, Map<String, List<Any>>> = fetchGET(url)
                val result = mutableMapOf<Pair<String, String>, Map<String, List<FavoriteDepartureInfo>>>()

                for ((key, deps) in response) {
                    val clean = key.removePrefix("stop_point:IDFM:").removePrefix("IDFM:")
                    val parts = clean.split(":")
                    if (parts.size >= 2) {
                        val stopId = parts[0]
                        val routeId = parts[1]
                        val sanitizedDeps = deps.mapValues { (_, items) ->
                            items.mapNotNull { item ->
                                when (item) {
                                    is Number -> FavoriteDepartureInfo(wait_sec = item.toInt())
                                    is String -> item.toIntOrNull()?.let { FavoriteDepartureInfo(wait_sec = it) }
                                    is Map<*, *> -> {
                                        val waitSec = (item["wait_sec"] as? Number)?.toInt() ?: 0
                                        val depSec = (item["dep_sec"] as? Number)?.toInt() ?: 0
                                        val schedDepSec = (item["scheduled_dep_sec"] as? Number)?.toInt() ?: 0
                                        val delaySec = (item["delay_sec"] as? Number)?.toInt() ?: 0
                                        val delayMin = (item["delay_minutes"] as? Number)?.toInt() ?: ((delaySec + 30) / 60).coerceAtLeast(0)
                                        val isDelayed = (item["is_delayed"] as? Boolean) ?: (delaySec >= 60)
                                        val timeStr = item["time_string"] as? String
                                        val schedTimeStr = item["scheduled_time_string"] as? String
                                        FavoriteDepartureInfo(
                                            wait_sec = waitSec,
                                            dep_sec = depSec,
                                            scheduled_dep_sec = schedDepSec,
                                            delay_sec = delaySec,
                                            delay_minutes = delayMin,
                                            is_delayed = isDelayed,
                                            time_string = timeStr,
                                            scheduled_time_string = schedTimeStr
                                        )
                                    }
                                    else -> null
                                }
                            }
                        }
                        result[stopId to routeId] = sanitizedDeps
                        result["IDFM:$stopId" to "IDFM:$routeId"] = sanitizedDeps
                    }
                }

                for (pair in list) {
                    val cleanPair = pair.first.removePrefix("IDFM:") to pair.second.removePrefix("IDFM:")
                    if (result.containsKey(cleanPair)) {
                        result[pair] = result[cleanPair]!!
                    }
                }
                result
            } catch (e: Exception) {
                e.printStackTrace()
                emptyMap()
            }
        }

    suspend fun getJourney(query: JourneyQuery): JourneyResponse = withContext(Dispatchers.IO) {
        val params = mutableListOf<Pair<String, String>>()

        when (val from = query.from) {
            is Position.Coordinates -> {
                params.add("from_lat" to from.lat.toString())
                params.add("from_lon" to from.lon.toString())
            }

            is Position.ByStop -> {
                params.add("from_stop" to from.stopId)
            }

            is Position.CurrentLocation -> {
                query.currentGps?.let {
                    params.add("from_lat" to it.first.toString())
                    params.add("from_lon" to it.second.toString())
                } ?: throw IllegalArgumentException("invalid 'from' position: CurrentLocation but currentGps is null")
            }
        }

        when (val to = query.to) {
            is Position.Coordinates -> {
                params.add("to_lat" to to.lat.toString())
                params.add("to_lon" to to.lon.toString())
            }

            is Position.ByStop -> {
                params.add("to_stop" to to.stopId)
            }

            is Position.CurrentLocation -> {
                query.currentGps?.let {
                    params.add("to_lat" to it.first.toString())
                    params.add("to_lon" to it.second.toString())
                } ?: throw IllegalArgumentException("invalid 'to' position: CurrentLocation but currentGps is null")
            }
        }

        if (!query.time.isNullOrBlank()) {
            params.add("time" to query.time)
            params.add("timetype" to if (query.timeType == TimeType.ARRIVAL) "arrival" else "departure")
        }

        if (!query.allowedModes.isNullOrEmpty()) {
            val modesStr = query.allowedModes.mapNotNull { mode ->
                when (mode) {
                    RouteType.TRAM -> "tram"
                    RouteType.METRO -> "metro"
                    RouteType.TRAIN -> "train"
                    RouteType.BUS -> "bus"
                    RouteType.FERRY -> "ferry"
                    RouteType.CABLE_TRAM -> "cabletram"
                    RouteType.AERIAL_LIFT -> "aeriallift"
                    RouteType.FUNICULAR -> "funicular"
                }
            }.joinToString(",")
            if (modesStr.isNotBlank()) {
                params.add("allowedmodes" to modesStr)
            }
        }

        if (query.withBike) {
            params.add("withbike" to "true")
        }
        if (query.wheelchair) {
            params.add("wheelchair" to "true")
        }

        val queryString = params.joinToString("&") { (key, value) ->
            "$key=${URLEncoder.encode(value, "UTF-8")}"
        }

        val url = "$BASE_URL/transit/journey?$queryString"
        fetchGET<JourneyResponse>(url)
    }

    suspend fun getLayerStops(x: Int, y: Int, z: Int): com.octara.maps.types.LayerStopsResponse {
        val url = "$BASE_URL/layers/stops/$x/$y/$z"
        return fetchGET<com.octara.maps.types.LayerStopsResponse>(url)
    }

    suspend fun getLayerRoutesLines(x: Int, y: Int, z: Int): com.octara.maps.types.LayerRoutesLinesResponse {
        val url = "$BASE_URL/layers/routesLines/$x/$y/$z"
        return fetchGET<com.octara.maps.types.LayerRoutesLinesResponse>(url)
    }

    suspend fun getJourney(
        from: Position,
        to: Position,
        time: String? = null,
        timeType: TimeType = TimeType.DEPARTURE,
        allowedModes: List<RouteType>? = null,
        withBike: Boolean = false,
        wheelchair: Boolean = false,
        currentGps: Pair<Double, Double>? = null
    ): JourneyResponse = getJourney(
        JourneyQuery(
            from = from,
            to = to,
            time = time,
            timeType = timeType,
            allowedModes = allowedModes,
            withBike = withBike,
            wheelchair = wheelchair,
            currentGps = currentGps
        )
    )
}

val api = Api()