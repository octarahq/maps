package com.octara.maps.types

import com.google.gson.annotations.SerializedName

data class LayerStopRoute(
    val name: String = "",
    val picto: String = "",
    val color: String = "",
    val type: RouteType? = null
)

data class LayerStop(
    val id: String = "",
    val name: String = "",
    val type: StopType = StopType.UNKNOWN,
    val lat: Double = 0.0,
    val lon: Double = 0.0,
    val routes: List<LayerStopRoute> = emptyList()
)

data class LayerStopsResponse(
    val layer: String = "",
    val total: Int = 0,
    val stops: List<LayerStop> = emptyList()
)

data class LayerRouteLine(
    @SerializedName("route_id") val routeId: String = "",
    val color: String = "",
    val points: List<List<Double>> = emptyList()
)

data class LayerRoutesLinesResponse(
    val layer: String = "",
    val total: Int = 0,
    val lines: List<LayerRouteLine> = emptyList()
)
