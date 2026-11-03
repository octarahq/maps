package com.octara.maps.ui.screens.map

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Place
import androidx.compose.material3.BottomSheetScaffold
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.rememberBottomSheetScaffoldState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.compose.foundation.background
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.layout.size
import com.octara.maps.navigation.LocalRouter
import com.octara.maps.ui.components.RouteIcon
import com.octara.maps.ui.components.RouteTypeIcon
import com.octara.maps.ui.theme.AppDimensions
import com.octara.maps.utils.Api

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun StopInfoScreen(
    stopId: String,
    modifier: Modifier = Modifier,
    onBack: (() -> Unit)? = null
) {
    val router = LocalRouter.current
    val scaffoldState = rememberBottomSheetScaffoldState()
    var isLoading by remember { mutableStateOf(true) }
    var stopInfo by remember { mutableStateOf<Api.StopInfoResponse?>(null) }

    LaunchedEffect(stopId) {
        isLoading = true
        stopInfo = com.octara.maps.utils.Api().getStopById(stopId)
        isLoading = false
    }

    BottomSheetScaffold(
        scaffoldState = scaffoldState,
        sheetShape = AppDimensions.BottomSheetShape,
        sheetPeekHeight = AppDimensions.BottomSheetPeekHeight,
        sheetShadowElevation = AppDimensions.BottomSheetElevation,
        sheetContent = {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 8.dp)
            ) {
                val info = stopInfo
                if (info == null && isLoading) {
                    Text(
                        text = "Chargement...",
                        style = MaterialTheme.typography.titleLarge,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                } else if (info == null) {
                    Text(
                        text = "Erreur lors du chargement de l'arrêt",
                        style = MaterialTheme.typography.titleLarge,
                        color = MaterialTheme.colorScheme.error
                    )
                } else {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(vertical = 8.dp),
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Box(
                            modifier = Modifier
                                .size(40.dp)
                                .clip(RoundedCornerShape(10.dp))
                                .background(MaterialTheme.colorScheme.primary.copy(alpha = 0.1f)),
                            contentAlignment = Alignment.Center
                        ) {
                            Icon(
                                imageVector = Icons.Default.Place,
                                contentDescription = "Icône de transport",
                                tint = MaterialTheme.colorScheme.primary,
                            )
                        }
                        Text(
                            text = info.stop.name,
                            style = MaterialTheme.typography.bodyLarge,
                            modifier = Modifier.padding(vertical = 8.dp)
                        )
                    }
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(vertical = 8.dp),
                    ) {
                        for ((routeType, routes) in (info.routes ?: emptyList()).groupBy { it.type }) {
                            Row(
                                modifier = Modifier
                                    .padding(horizontal = 2.dp, vertical = 2.dp),
                                horizontalArrangement = Arrangement.spacedBy(4.dp),
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Icon(
                                    imageVector = RouteTypeIcon(routeType = routeType),
                                    contentDescription = "route type",
                                    tint = MaterialTheme.colorScheme.primary,
                                )
                                for (route in routes.take(8)) {
                                    RouteIcon(route = route, modifier = Modifier.height(24.dp))
                                }
                                if (routes.size > 8) {
                                    Text(
                                        text = "+${routes.size - 8}",
                                        style = MaterialTheme.typography.labelLarge,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant
                                    )
                                }
                            }
                        }
                    }
                }
            }
        },
        modifier = modifier.fillMaxSize()
    ) {
        Box(modifier = Modifier.fillMaxSize()) {
            TransitMap(modifier = Modifier.fillMaxSize())
            IconButton(
                onClick = { onBack?.invoke() ?: router.goBack() },
                modifier = Modifier.align(Alignment.TopStart)
            ) {
                Icon(
                    imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                    contentDescription = "Retour",
                    tint = MaterialTheme.colorScheme.onSurface
                )
            }
        }
    }
}
